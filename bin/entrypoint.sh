#!/bin/bash

if [ "$RUN_MODE" = "mock" ]; then
  /bin/autoupdate-server-mock \
    -k /keys/private.key \
    -l 0.0.0.0:9999 \
    -p http://127.0.0.1:9999/ \
    -o getlantern \
    -n lantern \
    -repos getlantern/lantern,getlantern/beam
else
  /bin/autoupdate-server \
    -k /keys/private.key \
    -l 0.0.0.0:9999 \
    -p https://update.getlantern.org/ \
    -o getlantern \
    -r 0.3 \
    -n lantern \
    -repos getlantern/lantern,getlantern/beam
fi
