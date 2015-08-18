FROM quay.io/brianredbeard/corebox

ADD bin/dex-worker /opt/dex/bin/dex-worker
ADD bin/dex-overlord /opt/dex/bin/dex-overlord
ADD bin/dexctl /opt/dex/bin/dexctl

ENV DEX_WORKER_HTML_ASSETS /opt/dex/html/
ADD static/html/* $DEX_WORKER_HTML_ASSETS

ENV DEX_WORKER_EMAIL_ASSETS /opt/dex/email/
ADD static/email/* $DEX_WORKER_EMAIL_ASSETS
ADD static/fixtures/emailer.json.sample $DEX_WORKER_EMAIL_ASSETS/emailer.json
