#!/bin/bash

mkdir ${HOME}/codesign && cd ${HOME}/codesign
echo "${APPLE_CODE_SIGNING_CERTIFICATE_P12_BASE64}" | base64 --decode > cert.p12
curl 'https://www.apple.com/certificateauthority/DeveloperIDG2CA.cer' > apple.cer
KEYCHAIN_PATH=~/Library/Keychains/github_actions.keychain
security create-keychain -p "" ${KEYCHAIN_PATH}
security set-keychain-settings ${KEYCHAIN_PATH}
security unlock-keychain -u ${KEYCHAIN_PATH}
security list-keychains  -s ${KEYCHAIN_PATH} "${HOME}/Library/Keychains/login.keychain-db"
security import apple.cer -k ${KEYCHAIN_PATH} -T /usr/bin/codesign
security import cert.p12 -k ${KEYCHAIN_PATH} -P "${APPLE_CODE_SIGNING_CERTIFICATE_P12_PASSWORD}" -T /usr/bin/codesign
security set-key-partition-list -S apple-tool:,apple: -k "" ${KEYCHAIN_PATH}
rm -rf ${HOME}/codesign
cd ${GITHUB_WORKSPACE}
