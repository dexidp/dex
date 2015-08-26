# Deploying

Generate systemd unit files by injecting secrets into the unit file templates located in: `./static/...`.

```
source <path-to-secure>/prod/dex.env.txt
./build-units
```

