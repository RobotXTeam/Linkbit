# Linux Packaging

The release archive contains controller, relay, and agent binaries plus systemd install scripts under `deploy/`.

Typical backend install:

```bash
sudo mkdir -p /etc/linkbit
sudo cp deploy/controller.env.example /etc/linkbit/controller.env
sudo cp deploy/relay.env.example /etc/linkbit/relay.env
sudo ./deploy/install-controller.sh
sudo ./deploy/install-relay.sh
```

Edit `/etc/linkbit/*.env` before starting production services. Do not store plaintext API keys in the repository.
