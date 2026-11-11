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


## Testing the cashier camera on a phone

The cashier screen scans the voucher QR with the phone camera. Browsers allow the camera only on HTTPS, so
test it through the ngrok address, not `http://192.168.x.x`.

1. Start the demo and the tunnel (steps above). Open the `https://….ngrok-free.app/uz/login` address on the phone.
2. Sign in as the cashier, `+99891227300` (or `+998900000003` for Lotus Spa). Enter the code shown on the page, then the
   authenticator code if one is asked.
3. The phone opens the cashier screen and asks for camera permission. Allow it. The camera starts at once.
4. On a second phone or a printout, open a voucher from the buyer account (`+998900000001`, "Mening vauchelarim")
   and point the cashier camera at its QR code. The result card shows the deal, the masked buyer and the confirm button.
5. If the camera does not start: check that the address starts with `https://`, that the browser has camera permission
   for the site, and that no other app is using the camera. The code field still works without the camera.
