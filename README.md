# Guerra Academy Website Repository
This repository contains the code and infrastructure setup for the Guerra Academy website, accessible at https://guerraacademy.top.

Architecture Overview
Architecture - Guerra Academy

Please refer to the provided architecture diagram for an overview of the system architecture. The diagram illustrates the workflow from domain registration with GoDaddy to the live site, detailing the use of Cloudflare for CDN/SSL, hosting with WSO2 Choreo, and the technology stack, including Go (as the backend language), and SQLite for the Data layer.

# Build Instructions
To compile the website backend from source code, follow these steps:

bash
Copy code
cd front-end
go build -v ./...
sudo mkdir -p ${{ env.BACKEND_DIR }}
sudo service academy stop
sudo cp ./academy ${{ env.BACKEND_DIR }}
sudo cp -r ./templates ${{ env.BACKEND_DIR }}


# Prerequisites
The setup requires the following:

Nginx and Git must be installed on the server.
Go 1.19.2 should be downloaded and extracted in /usr/local.
The Go environment variable must be set to include its binaries in the PATH.
AWS CLI must be installed for any AWS-related operations.
The correct Nginx configuration must be copied to /etc/nginx/sites-enabled/default.
DNS records must be properly configured in Cloudflare.
For full installation and configuration details, consult the provided Ansible playbook (config-site.yml).
