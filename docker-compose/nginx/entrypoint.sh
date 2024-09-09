#!/bin/ash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

envsubst '
    ${MG_NGINX_SERVER_NAME}
    ${MG_AUTH_HTTP_PORT}
    ${MG_USERS_HTTP_PORT}
    ${MG_INVITATIONS_HTTP_PORT}' < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf

exec nginx -g "daemon off;"
