#!/bin/sh
# Swarm-CD entrypoint, used to import gpg keys if needed
if [ ! -z "${SOPS_GPG_PRIVATE_KEY_FILE}" ]; then
    echo "entrypoint.sh: importing gpg private key from file..."

    cat "${SOPS_GPG_PRIVATE_KEY_FILE}" | gpg --import

    if [ $? -ne 0 ]; then
        echo "entrypoint.sh: error: could not import GPG private key from file !"
        exit 1
    fi
    echo "entrypoint.sh: gpg key imported from file successfully."
elif [ ! -z "${SOPS_GPG_PRIVATE_KEY}" ]; then
    echo "entrypoint.sh: importing gpg private key from environment..."

    echo "${SOPS_GPG_PRIVATE_KEY}" | gpg --import
    
    if [ $? -ne 0 ]; then
        echo "entrypoint.sh: error: could not import GPG private key from environment !"
        exit 1
    fi
    echo "entrypoint.sh: gpg key imported from environment successfully."
else
    echo "entrypoint.sh: no gpg key found, skipping import"
fi

exec "$@"
