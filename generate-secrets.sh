#!/bin/bash

# 32 byte
ACCESS_SECRET=$(openssl rand -base64 32 | tr -d '\n')
echo "Access Secret: $ACCESS_SECRET"

# 64 byte
REFRESH_SECRET=$(openssl rand -base64 64 | tr -d '\n')
echo "Refresh Secret:"
echo "$REFRESH_SECRET"