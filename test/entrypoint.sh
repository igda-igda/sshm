#!/bin/bash

# Parse SSH_USERS environment variable
# Format: username:password:uid:gid:shell
if [ -n "$SSH_USERS" ]; then
    IFS=',' read -ra USERS <<< "$SSH_USERS"
    for user_info in "${USERS[@]}"; do
        IFS=':' read -ra USER_PARTS <<< "$user_info"
        if [ ${#USER_PARTS[@]} -eq 5 ]; then
            username="${USER_PARTS[0]}"
            password="${USER_PARTS[1]}"
            uid="${USER_PARTS[2]}"
            gid="${USER_PARTS[3]}"
            shell="${USER_PARTS[4]}"
            
            # Create group if it doesn't exist
            if ! getent group "$username" > /dev/null 2>&1; then
                addgroup -g "$gid" "$username"
            fi
            
            # Create user if it doesn't exist
            if ! getent passwd "$username" > /dev/null 2>&1; then
                adduser -D -u "$uid" -G "$username" -s "$shell" "$username"
                echo "$username:$password" | chpasswd
            fi
            
            # Create SSH directory for user
            mkdir -p "/home/$username/.ssh"
            chown "$username:$username" "/home/$username/.ssh"
            chmod 700 "/home/$username/.ssh"
            
            # Set up authorized_keys if it exists
            if [ -f "/home/$username/.ssh/authorized_keys" ]; then
                chown "$username:$username" "/home/$username/.ssh/authorized_keys"
                chmod 600 "/home/$username/.ssh/authorized_keys"
            fi
            
            echo "Created user: $username"
        fi
    done
fi

# Start SSH daemon
exec "$@"