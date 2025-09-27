#!/bin/sh
# Swarm-CD entrypoint, used to import gpg keys if needed
if [ ! -z "${SOPS_GPG_PRIVATE_KEY}" ]; then
    echo "entrypoint.sh: importing gpg private key..."

    echo "${SOPS_GPG_PRIVATE_KEY}" | gpg --import
    
    if [ $? -ne 0 ]; then
        echo "entrypoint.sh: error: could not import GPG private key !"
        exit 1
    fi
    echo "entrypoint.sh: gpg key imported successfully."
else
    echo "entrypoint.sh: no gpg key found, skipping import"
fi

# Call the app
if [ -z $@ ]; then
    echo "entrypoint.sh: starting SwarmCD..."
	/app/swarm-cd
else
	$@
fi
