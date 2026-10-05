<?php

declare(strict_types=1);

namespace Facinect\Identity\Http;

use Psr\Http\Message\ResponseInterface as Response;

final class JsonResponse
{
    public static function json(Response $response, array $data, int $status = 200): Response
    {
        $payload = json_encode($data, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE);
        $response->getBody()->write($payload === false ? '{}' : $payload);
        return $response
            ->withHeader('Content-Type', 'application/json; charset=utf-8')
            ->withStatus($status);
    }
}
