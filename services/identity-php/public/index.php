<?php

declare(strict_types=1);

use Facinect\Identity\App;
use Slim\Factory\AppFactory;

require __DIR__ . '/../vendor/autoload.php';

$app = AppFactory::create();
$app->addBodyParsingMiddleware();
$app->addRoutingMiddleware();
$error = $app->addErrorMiddleware(true, true, true);
$error->getDefaultErrorHandler()->forceContentType('application/json');

App::register($app);

$app->run();
