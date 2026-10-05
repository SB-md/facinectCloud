<?php

declare(strict_types=1);

namespace Facinect\Identity\Auth;

use Facinect\Identity\Db\Database;
use Facinect\Identity\Security\JwtService;
use PDO;

final class AuthService
{
    private Database $db;
    private JwtService $jwt;
    /** @var object|null Redis instance when ext-redis is available */
    private $redis = null;

    public function __construct(Database $db, JwtService $jwt)
    {
        $this->db = $db;
        $this->jwt = $jwt;
    }

    public function bootstrapAdminIfNeeded(): void
    {
        $email = trim((string) (getenv('BOOTSTRAP_ADMIN_EMAIL') ?: ''));
        $password = (string) (getenv('BOOTSTRAP_ADMIN_PASSWORD') ?: '');
        $name = trim((string) (getenv('BOOTSTRAP_ADMIN_NAME') ?: 'Admin'));
        if ($email === '' || $password === '') {
            return;
        }
        $stmt = $this->db->pdo()->prepare('SELECT id FROM users WHERE email = ? LIMIT 1');
        $stmt->execute([$email]);
        if ($stmt->fetch()) {
            return;
        }
        $hash = password_hash($password, PASSWORD_ARGON2ID);
        $ins = $this->db->pdo()->prepare(
            'INSERT INTO users (email, password_hash, full_name, status) VALUES (?, ?, ?, \'active\')'
        );
        $ins->execute([$email, $hash, $name]);
    }

    public function loginWithPassword(string $email, string $password, ?string $ip): array
    {
        if ($email === '' || $password === '') {
            throw new \InvalidArgumentException('email_and_password_required');
        }
        $stmt = $this->db->pdo()->prepare(
            'SELECT * FROM users WHERE email = ? AND status = \'active\' LIMIT 1'
        );
        $stmt->execute([$email]);
        $user = $stmt->fetch(PDO::FETCH_ASSOC);
        if (!$user || empty($user['password_hash']) || !password_verify($password, (string) $user['password_hash'])) {
            $this->audit(null, 'login_failed', $ip, ['email' => $email]);
            throw new \InvalidArgumentException('invalid_credentials');
        }
        $this->audit((int) $user['id'], 'login_password', $ip, null);
        return $this->issueTokenPair($user);
    }

    public function emailExists(string $email): bool
    {
        $email = strtolower(trim($email));
        if ($email === '') {
            return false;
        }
        $stmt = $this->db->pdo()->prepare('SELECT id FROM users WHERE email = ? LIMIT 1');
        $stmt->execute([$email]);
        return (bool) $stmt->fetch();
    }

    /** @param array{email:string,email_verified:bool,name:string,sub:string} $info */
    public function loginWithGoogle(array $info, ?string $ip): array
    {
        $email = strtolower(trim($info['email']));
        $sub = trim($info['sub']);
        $name = trim($info['name']) !== '' ? trim($info['name']) : $email;
        if ($email === '' || empty($info['email_verified'])) {
            throw new \InvalidArgumentException('email_not_verified');
        }

        $user = null;
        if ($sub !== '') {
            $stmt = $this->db->pdo()->prepare(
                'SELECT u.* FROM oauth_accounts o
                 INNER JOIN users u ON u.id = o.user_id
                 WHERE o.provider = \'google\' AND o.provider_sub = ? LIMIT 1'
            );
            $stmt->execute([$sub]);
            $user = $stmt->fetch(PDO::FETCH_ASSOC) ?: null;
        }
        if (!$user) {
            $stmt = $this->db->pdo()->prepare('SELECT * FROM users WHERE email = ? LIMIT 1');
            $stmt->execute([$email]);
            $user = $stmt->fetch(PDO::FETCH_ASSOC) ?: null;
        }
        if (!$user) {
            $ins = $this->db->pdo()->prepare(
                'INSERT INTO users (email, full_name, status) VALUES (?, ?, \'active\')'
            );
            $ins->execute([$email, $name]);
            $user = $this->userById((int) $this->db->pdo()->lastInsertId());
        }
        if (!$user || ($user['status'] ?? '') !== 'active') {
            throw new \InvalidArgumentException('unauthorized');
        }

        if ($sub !== '') {
            $link = $this->db->pdo()->prepare(
                'INSERT IGNORE INTO oauth_accounts (user_id, provider, provider_sub, email)
                 VALUES (?, \'google\', ?, ?)'
            );
            $link->execute([(int) $user['id'], $sub, $email]);
        }

        $this->audit((int) $user['id'], 'login_google', $ip, ['email' => $email]);
        return $this->issueTokenPair($user);
    }

    public function savePkce(string $state, string $verifier, string $redirectUri): void
    {
        $expires = (new \DateTimeImmutable('+10 minutes'))->format('Y-m-d H:i:s.v');
        $stmt = $this->db->pdo()->prepare(
            'INSERT INTO oauth_pkce (state, code_verifier, redirect_uri, expires_at)
             VALUES (?, ?, ?, ?)
             ON DUPLICATE KEY UPDATE code_verifier = VALUES(code_verifier),
               redirect_uri = VALUES(redirect_uri), expires_at = VALUES(expires_at)'
        );
        $stmt->execute([$state, $verifier, $redirectUri, $expires]);
    }

    /** @return array{code_verifier:string,redirect_uri:string}|null */
    public function consumePkce(string $state): ?array
    {
        $stmt = $this->db->pdo()->prepare('SELECT * FROM oauth_pkce WHERE state = ? LIMIT 1');
        $stmt->execute([$state]);
        $row = $stmt->fetch(PDO::FETCH_ASSOC);
        $del = $this->db->pdo()->prepare('DELETE FROM oauth_pkce WHERE state = ?');
        $del->execute([$state]);
        if (!$row) {
            return null;
        }
        if (strtotime((string) $row['expires_at']) < time()) {
            return null;
        }
        return [
            'code_verifier' => (string) $row['code_verifier'],
            'redirect_uri' => (string) $row['redirect_uri'],
        ];
    }

    public function createHandoff(array $tokenPayload): string
    {
        $code = bin2hex(random_bytes(24));
        $hash = hash('sha256', $code);
        $expires = (new \DateTimeImmutable('+2 minutes'))->format('Y-m-d H:i:s.v');
        $stmt = $this->db->pdo()->prepare(
            'INSERT INTO oauth_handoff (code_hash, payload_json, expires_at) VALUES (?, ?, ?)'
        );
        $stmt->execute([$hash, json_encode($tokenPayload), $expires]);
        return $code;
    }

    public function consumeHandoff(string $code): array
    {
        $hash = hash('sha256', $code);
        $stmt = $this->db->pdo()->prepare(
            'SELECT * FROM oauth_handoff WHERE code_hash = ? AND consumed_at IS NULL LIMIT 1'
        );
        $stmt->execute([$hash]);
        $row = $stmt->fetch(PDO::FETCH_ASSOC);
        if (!$row) {
            throw new \InvalidArgumentException('invalid_handoff');
        }
        if (strtotime((string) $row['expires_at']) < time()) {
            throw new \InvalidArgumentException('oauth_expired');
        }
        $upd = $this->db->pdo()->prepare('UPDATE oauth_handoff SET consumed_at = NOW(3) WHERE code_hash = ?');
        $upd->execute([$hash]);
        $payload = json_decode((string) $row['payload_json'], true);
        if (!is_array($payload)) {
            throw new \InvalidArgumentException('invalid_handoff');
        }
        return $payload;
    }

    public function requestOtp(string $phone): array
    {
        $phone = $this->normalizePhone($phone);
        if ($phone === '') {
            throw new \InvalidArgumentException('phone_required');
        }
        if ($this->otpRateLimited($phone)) {
            throw new \InvalidArgumentException('otp_rate_limited');
        }

        $code = (string) random_int(100000, 999999);
        $hash = hash('sha256', $phone . ':' . $code);
        $expires = (new \DateTimeImmutable('+5 minutes'))->format('Y-m-d H:i:s.v');

        $stmt = $this->db->pdo()->prepare(
            'INSERT INTO otp_challenges (phone_e164, code_hash, expires_at) VALUES (?, ?, ?)'
        );
        $stmt->execute([$phone, $hash, $expires]);

        // Local/dev: return OTP in response. Production WhatsApp sender replaces this.
        $out = [
            'ok' => true,
            'phone' => $phone,
            'expires_in' => 300,
            'delivery' => 'dev_inline',
        ];
        if ((getenv('APP_ENV') ?: 'local') === 'local') {
            $out['dev_otp'] = $code;
        }
        return $out;
    }

    public function verifyOtp(string $phone, string $code, ?string $ip): array
    {
        $phone = $this->normalizePhone($phone);
        $code = preg_replace('/\D+/', '', $code) ?? '';
        if ($phone === '' || strlen($code) < 4) {
            throw new \InvalidArgumentException('phone_and_code_required');
        }

        $stmt = $this->db->pdo()->prepare(
            'SELECT * FROM otp_challenges
             WHERE phone_e164 = ? AND consumed_at IS NULL
             ORDER BY id DESC LIMIT 1'
        );
        $stmt->execute([$phone]);
        $row = $stmt->fetch(PDO::FETCH_ASSOC);
        if (!$row) {
            throw new \InvalidArgumentException('otp_not_found');
        }
        if ((int) $row['attempts'] >= 5) {
            throw new \InvalidArgumentException('otp_locked');
        }
        if (strtotime((string) $row['expires_at']) < time()) {
            throw new \InvalidArgumentException('otp_expired');
        }

        $expect = hash('sha256', $phone . ':' . $code);
        if (!hash_equals((string) $row['code_hash'], $expect)) {
            $upd = $this->db->pdo()->prepare('UPDATE otp_challenges SET attempts = attempts + 1 WHERE id = ?');
            $upd->execute([(int) $row['id']]);
            throw new \InvalidArgumentException('otp_invalid');
        }

        $consume = $this->db->pdo()->prepare('UPDATE otp_challenges SET consumed_at = NOW(3) WHERE id = ?');
        $consume->execute([(int) $row['id']]);

        $user = $this->findOrCreateByPhone($phone);
        $this->audit((int) $user['id'], 'login_otp', $ip, ['phone' => $phone]);
        return $this->issueTokenPair($user);
    }

    public function refresh(string $refreshToken): array
    {
        if ($refreshToken === '') {
            throw new \InvalidArgumentException('refresh_token_required');
        }
        $hash = hash('sha256', $refreshToken);
        $stmt = $this->db->pdo()->prepare(
            'SELECT rt.*, u.* FROM refresh_tokens rt
             INNER JOIN users u ON u.id = rt.user_id
             WHERE rt.token_hash = ? AND rt.revoked_at IS NULL AND u.status = \'active\'
             LIMIT 1'
        );
        $stmt->execute([$hash]);
        $row = $stmt->fetch(PDO::FETCH_ASSOC);
        if (!$row) {
            throw new \InvalidArgumentException('invalid_refresh');
        }
        if (strtotime((string) $row['expires_at']) < time()) {
            throw new \InvalidArgumentException('refresh_expired');
        }

        // Rotate: revoke old, issue new pair
        $rev = $this->db->pdo()->prepare('UPDATE refresh_tokens SET revoked_at = NOW(3) WHERE id = ?');
        $rev->execute([(int) $row['id']]);

        $user = [
            'id' => $row['user_id'],
            'email' => $row['email'],
            'phone_e164' => $row['phone_e164'],
            'full_name' => $row['full_name'],
            'status' => $row['status'],
        ];
        return $this->issueTokenPair($user);
    }

    public function revokeRefresh(string $refreshToken): void
    {
        if ($refreshToken === '') {
            return;
        }
        $hash = hash('sha256', $refreshToken);
        $stmt = $this->db->pdo()->prepare(
            'UPDATE refresh_tokens SET revoked_at = NOW(3) WHERE token_hash = ? AND revoked_at IS NULL'
        );
        $stmt->execute([$hash]);
    }

    public function userById(int $id): ?array
    {
        $stmt = $this->db->pdo()->prepare('SELECT * FROM users WHERE id = ? LIMIT 1');
        $stmt->execute([$id]);
        $row = $stmt->fetch(PDO::FETCH_ASSOC);
        return $row ?: null;
    }

    public function publicUser(array $user): array
    {
        return [
            'id' => (int) $user['id'],
            'email' => $user['email'],
            'phone' => $user['phone_e164'],
            'full_name' => $user['full_name'],
            'status' => $user['status'],
        ];
    }

    private function issueTokenPair(array $user): array
    {
        $uid = (int) $user['id'];
        $access = $this->jwt->issueAccess($uid, [
            'name' => $user['full_name'] ?? '',
            'email' => $user['email'] ?? null,
            'phone' => $user['phone_e164'] ?? null,
        ]);

        $refreshRaw = bin2hex(random_bytes(32));
        $jti = $this->uuid();
        $hash = hash('sha256', $refreshRaw);
        $expires = (new \DateTimeImmutable('+' . $this->jwt->refreshTtl() . ' seconds'))
            ->format('Y-m-d H:i:s.v');

        $ins = $this->db->pdo()->prepare(
            'INSERT INTO refresh_tokens (user_id, jti, token_hash, expires_at) VALUES (?, ?, ?, ?)'
        );
        $ins->execute([$uid, $jti, $hash, $expires]);

        return [
            'token_type' => 'Bearer',
            'access_token' => $access,
            'expires_in' => $this->jwt->accessTtl(),
            'refresh_token' => $refreshRaw,
            'user' => $this->publicUser($user),
        ];
    }

    private function findOrCreateByPhone(string $phone): array
    {
        $stmt = $this->db->pdo()->prepare('SELECT * FROM users WHERE phone_e164 = ? LIMIT 1');
        $stmt->execute([$phone]);
        $user = $stmt->fetch(PDO::FETCH_ASSOC);
        if ($user) {
            return $user;
        }
        $ins = $this->db->pdo()->prepare(
            'INSERT INTO users (phone_e164, full_name, status) VALUES (?, ?, \'active\')'
        );
        $ins->execute([$phone, 'User ' . substr($phone, -4)]);
        $id = (int) $this->db->pdo()->lastInsertId();
        return $this->userById($id) ?? ['id' => $id, 'phone_e164' => $phone, 'email' => null, 'full_name' => '', 'status' => 'active'];
    }

    private function normalizePhone(string $phone): string
    {
        $digits = preg_replace('/\D+/', '', $phone) ?? '';
        if (strlen($digits) === 10) {
            $digits = '91' . $digits;
        }
        return $digits;
    }

    private function otpRateLimited(string $phone): bool
    {
        try {
            $redis = $this->redis();
            if ($redis === null) {
                return false;
            }
            $key = 'otp:rl:' . $phone;
            $count = (int) $redis->incr($key);
            if ($count === 1) {
                $redis->expire($key, 600);
            }
            return $count > 5;
        } catch (\Throwable $e) {
            return false;
        }
    }

    /** @return object|null */
    private function redis()
    {
        if ($this->redis !== null) {
            return $this->redis;
        }
        if (!class_exists('\Redis')) {
            return null;
        }
        try {
            $r = new \Redis();
            $host = getenv('REDIS_HOST') ?: '127.0.0.1';
            $port = (int) (getenv('REDIS_PORT') ?: 6379);
            if (!$r->connect($host, $port, 1.5)) {
                return null;
            }
            $this->redis = $r;
            return $this->redis;
        } catch (\Throwable $e) {
            return null;
        }
    }

    private function audit(?int $userId, string $event, ?string $ip, ?array $meta): void
    {
        $stmt = $this->db->pdo()->prepare(
            'INSERT INTO audit_log (user_id, event, ip, meta_json) VALUES (?, ?, ?, ?)'
        );
        $stmt->execute([
            $userId,
            $event,
            $ip,
            $meta === null ? null : json_encode($meta),
        ]);
    }

    private function uuid(): string
    {
        $data = random_bytes(16);
        $data[6] = chr((ord($data[6]) & 0x0f) | 0x40);
        $data[8] = chr((ord($data[8]) & 0x3f) | 0x80);
        return vsprintf('%s%s-%s-%s-%s-%s%s%s', str_split(bin2hex($data), 4));
    }
}
