<?php

declare(strict_types=1);

namespace Facinect\Identity\Security;

/**
 * Google OAuth 2.0 Authorization Code + PKCE (same flow as Facinect GoogleOAuthWeb).
 */
final class GoogleOAuth
{
    private const AUTH_URL = 'https://accounts.google.com/o/oauth2/v2/auth';
    private const TOKEN_URL = 'https://oauth2.googleapis.com/token';
    private const USERINFO_URL = 'https://www.googleapis.com/oauth2/v3/userinfo';

    private string $clientId;
    private string $clientSecret;
    private string $redirectUri;

    public function __construct(string $clientId, string $clientSecret, string $redirectUri)
    {
        $this->clientId = $clientId;
        $this->clientSecret = $clientSecret;
        $this->redirectUri = $redirectUri;
    }

    public static function fromEnv(): self
    {
        $id = trim((string) (getenv('GOOGLE_CLIENT_ID') ?: ''));
        $secret = trim((string) (getenv('GOOGLE_CLIENT_SECRET') ?: ''));
        $redirect = trim((string) (getenv('GOOGLE_OAUTH_REDIRECT_URI') ?: ''));
        if ($id === '' || $secret === '') {
            throw new \RuntimeException('oauth_config');
        }
        if ($redirect === '') {
            $redirect = 'http://localhost:8080/v1/auth/google/callback';
        }
        return new self($id, $secret, $redirect);
    }

    public function configured(): bool
    {
        return $this->clientId !== '' && $this->clientSecret !== '';
    }

    public function redirectUri(): string
    {
        return $this->redirectUri;
    }

    public static function generateCodeVerifier(): string
    {
        return rtrim(strtr(base64_encode(random_bytes(32)), '+/', '-_'), '=');
    }

    public static function codeChallenge(string $verifier): string
    {
        return rtrim(strtr(base64_encode(hash('sha256', $verifier, true)), '+/', '-_'), '=');
    }

    public static function generateState(): string
    {
        return bin2hex(random_bytes(16));
    }

    public function buildAuthorizationUrl(string $state, string $codeChallenge): string
    {
        $params = [
            'client_id' => $this->clientId,
            'redirect_uri' => $this->redirectUri,
            'response_type' => 'code',
            'scope' => 'openid email profile',
            'state' => $state,
            'code_challenge' => $codeChallenge,
            'code_challenge_method' => 'S256',
            'access_type' => 'online',
            'prompt' => 'select_account',
        ];
        return self::AUTH_URL . '?' . http_build_query($params);
    }

    /** @return array<string, mixed> */
    public function exchangeCode(string $code, string $codeVerifier): array
    {
        $body = http_build_query([
            'code' => $code,
            'client_id' => $this->clientId,
            'client_secret' => $this->clientSecret,
            'redirect_uri' => $this->redirectUri,
            'grant_type' => 'authorization_code',
            'code_verifier' => $codeVerifier,
        ]);
        $response = $this->httpPost(self::TOKEN_URL, $body);
        $decoded = json_decode($response, true);
        if (!is_array($decoded) || empty($decoded['access_token'])) {
            $err = (string) ($decoded['error'] ?? 'unknown');
            throw new \RuntimeException('oauth_exchange_failed:' . $err);
        }
        return $decoded;
    }

    /** @return array{email:string,email_verified:bool,name:string,sub:string} */
    public function fetchUserInfo(string $accessToken): array
    {
        $response = $this->httpGet(self::USERINFO_URL, $accessToken);
        $decoded = json_decode($response, true);
        if (!is_array($decoded) || empty($decoded['email'])) {
            throw new \RuntimeException('oauth_userinfo_failed');
        }
        return [
            'email' => (string) $decoded['email'],
            'email_verified' => filter_var($decoded['email_verified'] ?? false, FILTER_VALIDATE_BOOLEAN),
            'name' => (string) ($decoded['name'] ?? ''),
            'sub' => (string) ($decoded['sub'] ?? ''),
        ];
    }

    private function httpPost(string $url, string $body): string
    {
        if (function_exists('curl_init')) {
            $ch = curl_init($url);
            curl_setopt_array($ch, [
                CURLOPT_POST => true,
                CURLOPT_POSTFIELDS => $body,
                CURLOPT_RETURNTRANSFER => true,
                CURLOPT_TIMEOUT => 15,
                CURLOPT_HTTPHEADER => ['Content-Type: application/x-www-form-urlencoded'],
            ]);
            $response = curl_exec($ch);
            if ($response === false) {
                $err = curl_error($ch);
                curl_close($ch);
                throw new \RuntimeException('oauth_http:' . $err);
            }
            curl_close($ch);
            return (string) $response;
        }
        $ctx = stream_context_create([
            'http' => [
                'method' => 'POST',
                'header' => "Content-Type: application/x-www-form-urlencoded\r\n",
                'content' => $body,
                'timeout' => 15,
            ],
        ]);
        $response = @file_get_contents($url, false, $ctx);
        if ($response === false) {
            throw new \RuntimeException('oauth_http_failed');
        }
        return (string) $response;
    }

    private function httpGet(string $url, string $accessToken): string
    {
        if (function_exists('curl_init')) {
            $ch = curl_init($url);
            curl_setopt_array($ch, [
                CURLOPT_RETURNTRANSFER => true,
                CURLOPT_TIMEOUT => 15,
                CURLOPT_HTTPHEADER => ['Authorization: Bearer ' . $accessToken],
            ]);
            $response = curl_exec($ch);
            if ($response === false) {
                $err = curl_error($ch);
                curl_close($ch);
                throw new \RuntimeException('oauth_http:' . $err);
            }
            curl_close($ch);
            return (string) $response;
        }
        $ctx = stream_context_create([
            'http' => [
                'method' => 'GET',
                'header' => "Authorization: Bearer {$accessToken}\r\n",
                'timeout' => 15,
            ],
        ]);
        $response = @file_get_contents($url, false, $ctx);
        if ($response === false) {
            throw new \RuntimeException('oauth_userinfo_http_failed');
        }
        return (string) $response;
    }
}
