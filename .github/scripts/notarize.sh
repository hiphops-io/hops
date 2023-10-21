#!/bin/bash

echo "Submitting the app for notarization..."
NOTARY_OUT="$(xcrun notarytool submit "$1" --apple-id ${APPLE_LOGIN} --team-id ${APPLE_TEAM_ID} --password ${APPLE_NOTARYTOOL_PASSWORD} --wait)"

echo "$NOTARY_OUT"

# Extract Request UUID
REQUEST_UUID=$(echo "$NOTARY_OUT" | grep -oE -m 1 "id: [0-9a-f-]+" | awk '{print $2}')

if [ -z "$REQUEST_UUID" ]; then
  echo "Failed finding Request UUID in notarytool output"
  exit 1
fi

NOT_STATUS=$(echo "$NOTARY_OUT" | grep status: | awk -F ": " '{print $NF}')

if [[ "$NOT_STATUS" == Accepted* ]]; then
  echo "Notarization succeeded. Showing log"
else
  echo "Notarization failed with status: $NOT_STATUS. Showing log"
fi

xcrun notarytool log "$REQUEST_UUID" --apple-id ${APPLE_LOGIN} --team-id ${APPLE_TEAM_ID} --password ${APPLE_NOTARYTOOL_PASSWORD}

# Stapling is not supported for executables or zip files
# xcrun stapler staple -v zip/hops-${{ github.ref_name }}-${{ matrix.goos }}-${{ matrix.goarch }}.zip

echo "Notarization process is complete."
