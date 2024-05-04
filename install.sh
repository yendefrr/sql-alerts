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

# Verify installation
sqlal --v

# Change directory back to the original directory
cd ..

# Remove the cloned repository
rm -rf sql-alerts

echo "Installation completed."
