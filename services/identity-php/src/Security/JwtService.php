<?php

declare(strict_types=1);

namespace Facinect\Identity\Security;

use Firebase\JWT\JWT;
use Firebase\JWT\Key;

final class JwtService
{
    private string $privateKey;
    private string $publicKey;
    private string $issuer;
    private string $audience;
    private int $accessTtl;
    private int $refreshTtl;

    public function __construct(
        string $privateKey,
        string $publicKey,
        string $issuer,
        string $audience,
        int $accessTtl,
        int $refreshTtl
    ) {
        $this->privateKey = $privateKey;
        $this->publicKey = $publicKey;
        $this->issuer = $issuer;
        $this->audience = $audience;
        $this->accessTtl = $accessTtl;
        $this->refreshTtl = $refreshTtl;
    }

    public static function fromEnv(): self
    {
        $privPath = getenv('JWT_PRIVATE_KEY_PATH') ?: '/keys/jwt_private.pem';
        $pubPath = getenv('JWT_PUBLIC_KEY_PATH') ?: '/keys/jwt_public.pem';
        if (!is_readable($privPath) || !is_readable($pubPath)) {
            throw new \RuntimeException('JWT keys missing. Run scripts/generate-jwt-keys.sh');
        }
        return new self(
            (string) file_get_contents($privPath),
            (string) file_get_contents($pubPath),
            (string) (getenv('JWT_ISSUER') ?: 'https://api.facinect.local'),
            (string) (getenv('JWT_AUDIENCE') ?: 'facinect-apps'),
            (int) (getenv('JWT_ACCESS_TTL_SECONDS') ?: 900),
            (int) (getenv('JWT_REFRESH_TTL_SECONDS') ?: 2592000)
        );
    }

    public function accessTtl(): int
    {
        return $this->accessTtl;
    }

    public function refreshTtl(): int
    {
        return $this->refreshTtl;
    }

    public function issueAccess(int $userId, array $extra = []): string
    {
        $now = time();
        $payload = array_merge([
            'iss' => $this->issuer,
            'aud' => $this->audience,
            'iat' => $now,
            'nbf' => $now,
            'exp' => $now + $this->accessTtl,
            'sub' => (string) $userId,
            'typ' => 'access',
        ], $extra);
        return JWT::encode($payload, $this->privateKey, 'RS256', 'facinect-1');
    }

    public function parseAccess(string $token): array
    {
        $decoded = JWT::decode($token, new Key($this->publicKey, 'RS256'));
        $arr = (array) $decoded;
        if (($arr['typ'] ?? '') !== 'access') {
            throw new \InvalidArgumentException('not_access_token');
        }
        return $arr;
    }

    public function publicKeyPem(): string
    {
        return $this->publicKey;
    }

    /** Minimal JWKS for gateway / other services (single RSA key). */
    public function jwks(): array
    {
        $pub = openssl_pkey_get_public($this->publicKey);
        if ($pub === false) {
            return ['keys' => []];
        }
        $details = openssl_pkey_get_details($pub);
        if ($details === false || !isset($details['rsa'])) {
            return ['keys' => []];
        }
        $n = $this->base64Url((string) $details['rsa']['n']);
        $e = $this->base64Url((string) $details['rsa']['e']);
        return [
            'keys' => [[
                'kty' => 'RSA',
                'kid' => 'facinect-1',
                'use' => 'sig',
                'alg' => 'RS256',
                'n' => $n,
                'e' => $e,
            ]],
        ];
    }

    private function base64Url(string $bin): string
    {
        return rtrim(strtr(base64_encode($bin), '+/', '-_'), '=');
    }
}
