# Cloud Init

This directory contains the cloud-init configuration files for Cube AI.

## After the first boot

1. Generate access token from github to be able to pull the docker images and code from github
2. Login to the docker registry

```bash
docker login ghcr.io
```

Your username is your github username and your password is the access token you generated in step 1.

3. Clone the repository

```bash
git clone https://github.com/ultravioletrs/cube.git
```

Your username is your github username and your password is the access token you generated in step 1.

4. Pull the docker images

```bash
cd cube/docker-compose/
docker compose pull
```

5. Start the docker containers

```bash
docker compose up -d
```
