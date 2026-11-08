# Showing the demo to other people (ngrok)

The demo runs on your PC. To let someone else open it, put one HTTPS tunnel in front of the **web app**
(port 3000). The API is never exposed: the web server talks to it on `localhost:8080`.

Because ngrok gives an `https://` address, a phone can open it and the camera can scan voucher QR codes
(browsers allow the camera only on HTTPS or localhost).

## One time

1. Create a free account at <https://ngrok.com> and install ngrok from <https://ngrok.com/download>.
2. Copy your authtoken from the ngrok dashboard and run once:

   ```powershell
   ngrok config add-authtoken <your-token>
   ```

## Every time

1. Start the demo: `powershell -ExecutionPolicy Bypass -File scripts\demo.ps1`
2. In a second window start the tunnel: `ngrok http 3000`. Copy the `https://….ngrok-free.app` address it prints.
3. Tell the demo about that address (third window):

   ```powershell
   powershell -ExecutionPolicy Bypass -File scripts\share.ps1 -Url https://abc123.ngrok-free.app
   ```

   This writes `SITE_URL` and `ALLOWED_ORIGINS` into `.env.local`. They make sign-in, links, previews and
   the origin check accept the ngrok address.
4. Stop the demo (Ctrl+C) and start it again so it picks the new address up.
5. Send people `https://abc123.ngrok-free.app/uz`. On the free plan ngrok shows a one-time warning page
   first; they press **Visit Site**.

The free ngrok address changes every time you restart ngrok, so repeat steps 2–4 then. To go back to
localhost only, set `SITE_URL=http://localhost:3000` in `.env.local` and restart.

## If something does not work

- **Sign-in says the request is not allowed** — `ALLOWED_ORIGINS` does not contain the ngrok address.
  Run `scripts\share.ps1` again and restart the demo.
- **Images are missing** — restart the demo; images are served through the web app, so the tunnel is enough.
- **Anyone with the link can use the demo** — it is a demo with no real data, but stop ngrok when you are done.
