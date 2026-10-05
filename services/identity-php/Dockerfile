FROM composer:2 AS vendor
WORKDIR /app
COPY composer.json ./
RUN composer install --no-dev --prefer-dist --no-interaction --no-scripts --optimize-autoloader

FROM php:8.2-cli-alpine
RUN apk add --no-cache curl-dev \
    && docker-php-ext-install pdo_mysql curl

WORKDIR /app
COPY --from=vendor /app/vendor ./vendor
COPY . .

ENV PORT=8080
EXPOSE 8080
CMD ["php", "-S", "0.0.0.0:8080", "-t", "public", "public/index.php"]
