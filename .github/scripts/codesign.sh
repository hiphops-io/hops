#!/bin/bash

codesign -s "${APPLE_CODE_SIGNING_DEVELOPER_ID}" \
  --force --options runtime --timestamp \
  --entitlements platforms/macos/hops.entitlements \
  "$1"

echo "Verifying the signature..."
codesign -vvvv --deep --strict "$1"
if [ $? -eq 0 ]; then
    echo "The app has been signed successfully."
else
    echo "The signature is not valid."
    exit 1
fi
