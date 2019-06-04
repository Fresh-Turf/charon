# Charon

![Charon](charon.jpg)

A tiny WebSocket server that broadcasts messages from Redis pub/sub channel.

# You may ask why

It's built specifically to address the inability (or unnecessary difficulty) of a Python Flask app (swh-cv) to handle both multithreading and websockets simultaneously while running on Gunicorn application server. Since the said app is already utilising Redis pub/sub to communicate with Sentinel, it seemed reasonable to use another channel on the same Redis instance to broadcast progress updates. Websockets are used to push these updates to the front end.

# Install

1. If doing for the first time: `glide install` and `cp .env.example .env`, then edit `.env` as needed.
2. `make && sudo make install`
3. `sudo service charon status`

Note that make install will only work on Linux

# Uninstall

`sudo make uninstall`

# Credits

Forked from https://github.com/connoryates/websocket-redis