#!/bin/bash

USER_NAME=$(whoami)
USER_ID=$(id -u "$USER_NAME")
USER_GROUP=$(id -g "$USER_NAME")

echo "Username: $USER_NAME"
echo "UID: $USER_ID"
echo "GID: $USER_GROUP"
