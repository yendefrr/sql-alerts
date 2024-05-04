#!/bin/bash

# Clone the repository
git clone https://github.com/yendefrr/sql-alerts.git sql-alerts

# Check if cloning was successful
if [ $? -eq 0 ]; then
    echo "Repository cloned successfully."
else
    echo "Failed to clone repository. Exiting."
    exit 1
fi

# Change directory to the cloned repository
cd sql-alerts

# Install sqlal
go install ./cmd/sqlal

directory_to_add="/root/go/bin"

# Check if the directory is already in the PATH
if [[ ":$PATH:" != *":$directory_to_add:"* ]]; then
    # Add the directory to the PATH in the appropriate shell configuration file
    if [[ $SHELL == "/bin/bash" ]]; then
        echo 'export PATH="$PATH:'"$directory_to_add"'"' >> ~/.bashrc
        source ~/.bashrc
    elif [[ $SHELL == "/bin/zsh" ]]; then
        echo 'export PATH="$PATH:'"$directory_to_add"'"' >> ~/.zshrc
        source ~/.zshrc
    else
        echo "Unsupported shell: $SHELL"
        exit 1
    fi
    echo "Directory added to PATH: $directory_to_add"
else
    echo "Directory already exists in PATH: $directory_to_add"
fi

# Verify installation
sqlal --v

# Change directory back to the original directory
cd ..

# Remove the cloned repository
rm -rf sql-alerts

echo "Installation completed."
