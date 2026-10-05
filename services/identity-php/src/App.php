<?php

declare(strict_types=1);

namespace Facinect\Identity;

use Facinect\Identity\Auth\AuthService;
use Facinect\Identity\Db\Database;
use Facinect\Identity\Db\Schema;
use Facinect\Identity\Http\JsonResponse;
use Facinect\Identity\Security\GoogleOAuth;
use Facinect\Identity\Security\JwtService;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Slim\App as SlimApp;

final class App
{
    public static function register(SlimApp $slim): void
    {
        $db = Database::fromEnv();
        Schema::ensure($db->pdo());
        $jwt = JwtService::fromEnv();
        $auth = new AuthService($db, $jwt);
        $auth->bootstrapAdminIfNeeded();

        $slim->options('/{routes:.+}', function (Request $request, Response $response): Response {
            return self::withCors($response);
        });

        $slim->add(function (Request $request, $handler) {
            $response = $handler->handle($request);
            return self::withCors($response);
        });

        $serveLoginUi = function (Request $request, Response $response): Response {
            $path = dirname(__DIR__) . '/public/ui/index.html';
            if (!is_readable($path)) {
                return JsonResponse::json($response, ['error' => 'ui_missing'], 404);
            }
            $html = (string) file_get_contents($path);
            $response->getBody()->write($html);
            return $response->withHeader('Content-Type', 'text/html; charset=utf-8');
        };
        $slim->get('/login[/]', $serveLoginUi);
        $slim->get('/v1/auth/ui[/]', $serveLoginUi);

        $slim->get('/v1/auth/health', function (Request $request, Response $response) use ($db): Response {
            $ok = $db->ping();
            $googleOk = trim((string) (getenv('GOOGLE_CLIENT_ID') ?: '')) !== ''
                && trim((string) (getenv('GOOGLE_CLIENT_SECRET') ?: '')) !== '';
            return JsonResponse::json($response, [
                'service' => 'identity',
                'status' => $ok ? 'ok' : 'degraded',
                'mysql' => $ok,
                'google_oauth' => $googleOk,
                'time' => gmdate('c'),
            ], $ok ? 200 : 503);
        });

        $slim->post('/v1/auth/check-email', function (Request $request, Response $response) use ($auth): Response {
            $body = (array) $request->getParsedBody();
            $email = strtolower(trim((string) ($body['email'] ?? '')));
            if ($email === '' || !filter_var($email, FILTER_VALIDATE_EMAIL)) {
                return JsonResponse::json($response, ['error' => 'invalid_email'], 400);
            }
            // Local identity: allow password step for any email (Google users may not have password).
            // Mirror Facinect UX: exist=true shows password; unknown still can use Google.
            $exists = $auth->emailExists($email);
            return JsonResponse::json($response, [
                'exists' => $exists,
                'can_password' => $exists,
            ]);
        });

        $slim->post('/v1/auth/login', function (Request $request, Response $response) use ($auth): Response {
            $body = (array) $request->getParsedBody();
            $email = trim((string) ($body['email'] ?? ''));
            $password = (string) ($body['password'] ?? '');
            $ip = $request->getServerParams()['REMOTE_ADDR'] ?? null;
            try {
                $tokens = $auth->loginWithPassword($email, $password, is_string($ip) ? $ip : null);
                return JsonResponse::json($response, $tokens);
            } catch (\InvalidArgumentException $e) {
                return JsonResponse::json($response, ['error' => $e->getMessage()], 401);
            }
        });

        $slim->get('/v1/auth/google/start', function (Request $request, Response $response) use ($auth): Response {
            try {
                $google = GoogleOAuth::fromEnv();
            } catch (\Throwable $e) {
                return $response
                    ->withHeader('Location', '/login?error=oauth_config')
                    ->withStatus(302);
            }
            $state = GoogleOAuth::generateState();
            $verifier = GoogleOAuth::generateCodeVerifier();
            $challenge = GoogleOAuth::codeChallenge($verifier);
            $auth->savePkce($state, $verifier, $google->redirectUri());
            $url = $google->buildAuthorizationUrl($state, $challenge);
            return $response->withHeader('Location', $url)->withStatus(302);
        });

        $slim->get('/v1/auth/google/callback', function (Request $request, Response $response) use ($auth): Response {
            $params = $request->getQueryParams();
            if (!empty($params['error'])) {
                $code = ($params['error'] === 'access_denied') ? 'oauth_denied' : 'oauth_failed';
                return $response
                    ->withHeader('Location', '/login?error=' . rawurlencode($code))
                    ->withStatus(302);
            }
            $code = trim((string) ($params['code'] ?? ''));
            $state = trim((string) ($params['state'] ?? ''));
            if ($code === '' || $state === '') {
                return $response
                    ->withHeader('Location', '/login?error=invalid_state')
                    ->withStatus(302);
            }
            $pkce = $auth->consumePkce($state);
            if ($pkce === null) {
                return $response
                    ->withHeader('Location', '/login?error=oauth_expired')
                    ->withStatus(302);
            }
            try {
                $google = GoogleOAuth::fromEnv();
                if ($pkce['redirect_uri'] !== $google->redirectUri()) {
                    return $response
                        ->withHeader('Location', '/login?error=oauth_redirect_mismatch')
                        ->withStatus(302);
                }
                $tokenData = $google->exchangeCode($code, $pkce['code_verifier']);
                $info = $google->fetchUserInfo((string) $tokenData['access_token']);
                $ip = $request->getServerParams()['REMOTE_ADDR'] ?? null;
                $tokens = $auth->loginWithGoogle($info, is_string($ip) ? $ip : null);
                $handoff = $auth->createHandoff($tokens);
                return $response
                    ->withHeader('Location', '/login?google=1&handoff=' . rawurlencode($handoff))
                    ->withStatus(302);
            } catch (\InvalidArgumentException $e) {
                return $response
                    ->withHeader('Location', '/login?error=' . rawurlencode($e->getMessage()))
                    ->withStatus(302);
            } catch (\Throwable $e) {
                $msg = $e->getMessage();
                $err = 'oauth_failed';
                if (str_contains($msg, 'oauth_exchange_failed:invalid_client')) {
                    $err = 'oauth_bad_secret';
                } elseif (str_contains($msg, 'oauth_exchange_failed:redirect_uri_mismatch')) {
                    $err = 'oauth_redirect_mismatch';
                } elseif (str_contains($msg, 'oauth_config')) {
                    $err = 'oauth_config';
                }
                return $response
                    ->withHeader('Location', '/login?error=' . rawurlencode($err))
                    ->withStatus(302);
            }
        });

        $slim->post('/v1/auth/google/handoff', function (Request $request, Response $response) use ($auth): Response {
            $body = (array) $request->getParsedBody();
            $handoff = trim((string) ($body['handoff'] ?? ''));
            try {
                $tokens = $auth->consumeHandoff($handoff);
                return JsonResponse::json($response, $tokens);
            } catch (\InvalidArgumentException $e) {
                return JsonResponse::json($response, ['error' => $e->getMessage()], 401);
            }
        });

        $slim->post('/v1/auth/otp/request', function (Request $request, Response $response) use ($auth): Response {
            $body = (array) $request->getParsedBody();
            $phone = trim((string) ($body['phone'] ?? $body['whatsappNo'] ?? ''));
            try {
                $result = $auth->requestOtp($phone);
                return JsonResponse::json($response, $result);
            } catch (\InvalidArgumentException $e) {
                return JsonResponse::json($response, ['error' => $e->getMessage()], 400);
            }
        });

        $slim->post('/v1/auth/otp/verify', function (Request $request, Response $response) use ($auth): Response {
            $body = (array) $request->getParsedBody();
            $phone = trim((string) ($body['phone'] ?? $body['whatsappNo'] ?? ''));
            $code = trim((string) ($body['code'] ?? $body['otp'] ?? ''));
            $ip = $request->getServerParams()['REMOTE_ADDR'] ?? null;
            try {
                $tokens = $auth->verifyOtp($phone, $code, is_string($ip) ? $ip : null);
                return JsonResponse::json($response, $tokens);
            } catch (\InvalidArgumentException $e) {
                return JsonResponse::json($response, ['error' => $e->getMessage()], 401);
            }
        });

        $slim->post('/v1/auth/token/refresh', function (Request $request, Response $response) use ($auth): Response {
            $body = (array) $request->getParsedBody();
            $refresh = (string) ($body['refresh_token'] ?? '');
            try {
                $tokens = $auth->refresh($refresh);
                return JsonResponse::json($response, $tokens);
            } catch (\InvalidArgumentException $e) {
                return JsonResponse::json($response, ['error' => $e->getMessage()], 401);
            }
        });

        $slim->post('/v1/auth/logout', function (Request $request, Response $response) use ($auth): Response {
            $body = (array) $request->getParsedBody();
            $refresh = (string) ($body['refresh_token'] ?? '');
            $auth->revokeRefresh($refresh);
            return JsonResponse::json($response, ['ok' => true]);
        });

        $slim->get('/v1/auth/me', function (Request $request, Response $response) use ($auth, $jwt): Response {
            $header = $request->getHeaderLine('Authorization');
            if (!preg_match('/^Bearer\s+(\S+)$/i', $header, $m)) {
                return JsonResponse::json($response, ['error' => 'missing_bearer'], 401);
            }
            try {
                $claims = $jwt->parseAccess($m[1]);
                $user = $auth->userById((int) ($claims['sub'] ?? 0));
                if ($user === null) {
                    return JsonResponse::json($response, ['error' => 'user_not_found'], 401);
                }
                return JsonResponse::json($response, [
                    'user' => $auth->publicUser($user),
                    'claims' => [
                        'sub' => $claims['sub'] ?? null,
                        'iss' => $claims['iss'] ?? null,
                        'aud' => $claims['aud'] ?? null,
                        'exp' => $claims['exp'] ?? null,
                    ],
                ]);
            } catch (\Throwable $e) {
                return JsonResponse::json($response, ['error' => 'invalid_token'], 401);
            }
        });

        $slim->get('/v1/auth/jwks.json', function (Request $request, Response $response) use ($jwt): Response {
            return JsonResponse::json($response, $jwt->jwks());
        });
    }

    private static function withCors(Response $response): Response
    {
        $origin = getenv('CORS_ORIGINS') ?: '*';
        return $response
            ->withHeader('Access-Control-Allow-Origin', $origin)
            ->withHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization')
            ->withHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
    }
}
