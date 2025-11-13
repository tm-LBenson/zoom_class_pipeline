# Zoom Class Pipeline

A small, cross‑platform toolchain that turns your local Zoom recordings into a simple “class replays” page:

- Watches a folder of Zoom **local recordings**.
- Uploads new `.mp4` files to your own **Amazon S3** bucket.
- Keeps a `recordings.json` index in S3 with metadata (date, level, URL).
- Serves videos and JSON through **CloudFront**.
- Renders a React web app where students can browse and play recordings by date and level.

This repository contains both the backend uploader (Go CLI) and the frontend viewer (Vite + React).

---

## Features

- **Automated upload** – run one command (or a scheduled task) after class to push new recordings.
- **Cheap storage** – use your own S3 bucket and CloudFront distribution.
- **Level‑aware index** – a single `recordings.json` file that tracks multiple levels (e.g. “Level 1”, “Level 2”) and multiple cohorts.
- **Replay UI** – search box, per‑level filter, and an embedded HTML5 video player.

The structure and install scripts follow the same pattern as the `big-log-viewer` project.

---

## Repository layout

- `main.go` – Go CLI that scans a folder, uploads videos to S3, and updates `recordings.json`.
- `config.json` – Configuration (created on first run).
- `frontend/` – React app for browsing and playing recordings.
- `scripts/` – Optional helper install scripts for macOS, Linux, and Windows.

---

## Prerequisites

### Accounts and services

Each instructor needs:

- A **Zoom** account with local recording enabled (Pro or better).
- An **AWS account** with permission to create:
  - one S3 bucket (for video + JSON),
  - one CloudFront distribution in front of that bucket,
  - one IAM user with programmatic access to that bucket.

### Local tools

You can use the pre‑made install scripts (see [Install the uploader](#install-the-uploader)) or install tools manually:

- **Git**
- **Go** 1.21+ (CLI)
- For building the frontend: **Node.js** 20+ and `npm`

---

## 1. Configure Zoom local recording

1. Open the Zoom desktop app.
2. Go to **Settings > Recording**.
3. Turn on **Store my recording at:** and pick a stable folder, e.g.:

   - Windows: `C:\Users\you\Documents\Zoom`
   - macOS: `/Users/you/Documents/Zoom`
   - Linux: `/home/you/Zoom` (or similar)

4. In the Zoom web portal, edit your recurring class meeting:
   - Enable **Automatically record meeting** > **On the local computer** (Optional).

You will point the uploader at this folder via the `watchDir` setting in `config.json`.

---

## 2. AWS setup

### 2.1 Create an S3 bucket

1. In the AWS console, open **S3 > Create bucket**.
2. Bucket name: something unique like `codex-recordings-yourname`.
3. Region: pick a region close to you (e.g. `us-east-1`).
4. For a simple start, uncheck **Block all public access**.
5. Click **Create bucket**.

### 2.2 Allow public read of objects (optional simple mode)

On the bucket’s **Permissions** tab, set a bucket policy:
You will see the Resource above the textarea for the JSON.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicReadForReplays",
      "Effect": "Allow",
      "Principal": "*",
      "Action": ["s3:GetObject"],
      "Resource": "arn:aws:s3:::codex-recordings-yourname/*"
    }
  ]
}
```

Replace `codex-recordings-yourname` with your bucket name.

> If you prefer private buckets with CloudFront Origin Access Control, you can set that up later. For internal class use, a public read bucket behind CloudFront is usually fine.

### 2.3 Configure CORS

On the same **Permissions** tab, set **CORS** so browsers can fetch `recordings.json`:

```json
[
  {
    "AllowedHeaders": ["*"],
    "AllowedMethods": ["GET", "HEAD"],
    "AllowedOrigins": ["*"],
    "ExposeHeaders": []
  }
]
```

Later you can replace `*` with your frontend origin (e.g. `"https://your-class-recording.netlify.app"`) (Recommended).

### 2.4 Create an IAM “service account” user

1. Go to **IAM > Users > Create user**.
2. Name: `codex-recorder-yourname`.
3. Check **Provide user access to the AWS Management Console – optional** off (CLI only).
4. On “Permissions”, attach a custom policy that can read/write this bucket:

   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": ["s3:PutObject", "s3:GetObject", "s3:ListBucket"],
         "Resource": [
           "arn:aws:s3:::codex-recordings-yourname",
           "arn:aws:s3:::codex-recordings-yourname/*"
         ]
       }
     ]
   }
   ```

5. After creating the user, add an **Access key** (programmatic access) and save:
   - Access key ID
   - Secret access key

You will paste these into `config.json` as `awsAccessKeyId` and `awsSecretAccessKey`.

### 2.5 Create a CloudFront distribution

1. Go to **CloudFront > Distributions > Create distribution**.
2. Origin domain: pick your S3 bucket (`codex-recordings-yourname`).
3. Viewer protocol policy: **Redirect HTTP to HTTPS**.
4. Leave the other defaults, click **Create distribution**.
5. After it finishes deploying, note the **Domain name**, e.g. `d123abcxyz.cloudfront.net`.

This will be your `baseUrl` in `config.json`.

---

## 3. Install the uploader

You can either run it from source (useful for development) or let a script build and place a single binary for you.

### 3.1 macOS (install script)

From **Terminal**:

```bash
curl -L https://raw.githubusercontent.com/tm-LBenson/zoom_class_pipeline/main/scripts/install_recorder_mac.sh -o install_recorder_mac.sh
chmod +x ./install_recorder_mac.sh
./install_recorder_mac.sh
```

The script will:

- Ensure `git` and `go` exist (via Homebrew if needed),
- Clone this repo into a temp directory,
- Build a `zoom-recorder` binary,
- Place it at `~/Desktop/zoom-recorder`.

### 3.2 Linux (install script)

From your shell:

```bash
curl -L https://raw.githubusercontent.com/tm-LBenson/zoom_class_pipeline/main/scripts/install_recorder_linux.sh -o install_recorder_linux.sh
chmod +x ./install_recorder_linux.sh
./install_recorder_linux.sh
```

The script will:

- Check for `git` and `go` and abort with a message if they are missing,
- Clone this repo into a temp directory,
- Build a `zoom-recorder` binary,
- Place it at `~/zoom-recorder/zoom-recorder`.

### 3.3 Windows (install script)

From **PowerShell (Run as Administrator)**:

```powershell
Invoke-WebRequest -Uri https://raw.githubusercontent.com/tm-LBenson/zoom_class_pipeline/main/scripts/install_recorder_windows.ps1 -OutFile install_recorder_windows.ps1
Set-ExecutionPolicy Bypass -Scope Process -Force
.\install_recorder_windows.ps1
```

The script will:

- Ensure `git` and `go` exist (using `winget` if necessary),
- Clone this repo into `%TEMP%\zoom_class_pipeline`,
- Build `zoom-recorder.exe`,
- Copy it to your Desktop.

### 3.4 Manual build (any OS)

If you prefer to build manually:

```bash
git clone https://github.com/tm-LBenson/zoom_class_pipeline.git
cd zoom_class_pipeline
go build -o zoom-recorder
```

On Windows:

```powershell
git clone https://github.com/tm-LBenson/zoom_class_pipeline.git
cd .\zoom_class_pipeline
go build -o zoom-recorder.exe
```

---

## 4. Configure `config.json`

Run the uploader once to generate a default config:

```bash
./zoom-recorder
```

You should see a message like “config file created at config.json, fill in values and run again”.

Edit `config.json` and set:

```json
{
  "watchDir": "C:UsersyouDocumentsZoom",
  "bucket": "codex-recordings-yourname",
  "region": "us-east-1",
  "videoPrefix": "level1",
  "indexKey": "recordings.json",
  "baseUrl": "https://d123abcxyz.cloudfront.net",
  "topicPrefix": "Level 1",
  "awsAccessKeyId": "AKIA...",
  "awsSecretAccessKey": "..."
}
```

Notes:

- `watchDir` must match your Zoom local recording folder.
- `videoPrefix` is the subfolder in S3 (one per level, e.g. `level1`, `level2`).
- `topicPrefix` is the human label for this level and is stored as `level` in `recordings.json`.
- `indexKey` can stay `recordings.json` so all levels share one index file.

Make sure `config.json` is in `.gitignore` so you don’t commit credentials.

---

## 5. Run the uploader

After each class (or after you have some test recordings):

```bash
./zoom-recorder
```

The tool will:

1. Read `config.json`.
2. Scan `watchDir` for `.mp4` files that are at least ~30 seconds old.
3. For each new file:
   - upload to `s3://bucket/videoPrefix/YYYY/MM/DD/filename.mp4`,
   - compute a public URL based on `baseUrl`,
   - append an entry to `recordings.json` with fields: `id`, `level`, `topic`, `start`, `duration`, `link`, `file`.
4. Write `recordings.json` back to S3 at `indexKey`.

If CloudFront seems to cache an old version of `recordings.json`, create an invalidation:

1. CloudFront > your distribution > **Invalidations > Create invalidation**.
2. Path: `/recordings.json`.
3. Wait until the status is **Completed**, then refresh the frontend.

---

## 6. Schedule the uploader (optional but recommended)

### 6.1 Windows (Task Scheduler)

1. Build or install `zoom-recorder.exe` as above.
2. Open **Task Scheduler > Create Task**.
3. General:
   - Name: `Zoom Recorder`
   - “Run whether user is logged on or not”
4. Trigger:
   - New > Daily, repeat every 1 day.
   - Start at e.g. `22:10` (10 minutes after your last class ends).
   - Optionally limit to specific days.
5. Action:
   - “Start a program”
   - Program/script: `C:\Users\you\Desktop\zoom-recorder.exe`
   - Start in: folder where `config.json` lives.
6. Click OK. Enter your password when prompted so Windows can run it unattended.

### 6.2 macOS (cron)

1. Ensure `zoom-recorder` and `config.json` are in a stable folder, e.g. `~/zoom-recorder`.
2. Edit your crontab:

   ```bash
   crontab -e
   ```

3. Add a line:

   ```text
   10 22 * * 1-5 cd /Users/you/zoom-recorder && ./zoom-recorder >> uploader.log 2>&1
   ```

4. Save and exit. This runs the uploader at 22:10 Monday–Friday.

### 6.3 Linux (cron)

1. Place `zoom-recorder` and `config.json` in e.g. `/opt/zoom-recorder`.
2. Edit your crontab:

   ```bash
   crontab -e
   ```

3. Add:

   ```text
   10 22 * * 1-5 cd /opt/zoom-recorder && ./zoom-recorder >> /var/log/zoom-recorder.log 2>&1
   ```

4. Save. Cron will run it each weekday night.

---

## 7. Frontend (class replay viewer)

The frontend lives under `frontend/` and is a Vite + React app that:

- fetches `recordings.json` from CloudFront (via `VITE_FEED_URL` or a `?feed=` query),
- shows a vertical list of recordings with:
  - level + date/time,
  - search box,
  - level filter (via the kebab menu),
- plays the selected video in an HTML5 `<video>` element,
- supports light/dark mode.

### 7.1 Local development

From `frontend/`:

```bash
npm install
npm run dev
```

By default the app will try to load `recordings.json` from the same host. For local testing, copy your `recordings.json` next to `frontend/public/recordings.json` or run the dev server with a `?feed=` query:

```text
http://localhost:5173/?feed=https://d123abcxyz.cloudfront.net/recordings.json
```

> if you did not setup CORS, you will need to clear cloudFront cache with invalidation on the resource. (You can use /\* for the path when you create a new invalidation)

### 7.2 Build for static hosting

From `frontend/`:

```bash
npm run build
```

The Vite config is set to emit static files into `dist`. That folder will contain:

- `index.html`
- `assets/…`

### 7.3 Deploy with Netlify

1. Push your repo to GitHub.
2. In Netlify:

   - New site from Git,
   - Choose the repo,
   - Base: `/frontend/`
   - Build command: `npm install && npm run build`,

3. In Netlify **Site settings > Build & deploy > Environment > Edit variables**:
   - Add `VITE_FEED_URL` with value `https://d123abcxyz.cloudfront.net/recordings.json`.
4. Deploy. Once complete, open your Netlify URL; you should see your recordings.

### 8. Multiple levels and cohorts

`recordings.json` is a flat array; each entry has a `level` field and a `topic`.

To record for a new level:

1. Stop the uploader.
2. Edit `config.json`:
   - change `videoPrefix` to a new folder name, e.g. `level2`,
   - change `topicPrefix` to `"Level 2"`,
   - leave `indexKey` as `"recordings.json"`.
3. Run the uploader after your Level 2 class.

New entries will have `"level": "Level 2"` and URLs under `/level2/...`. The frontend will show both levels and let you filter with the kebab menu.

If you want to separate cohorts (e.g. `2025-01` vs `2025-03`), you can include the cohort name in `topicPrefix` or encode it into `videoPrefix` (e.g. `level1-jan2025`). The UI will still treat `level` as whatever you set in `topicPrefix`.

---

## 9. Removing or correcting recordings

To remove or fix one recording:

1. **Remove or replace the video:**
   - In S3, delete or overwrite the specific `.mp4` under your `videoPrefix` path.
2. **Update the index:**
   - Either:
     - run a small script to regenerate `recordings.json`, or
     - manually edit `recordings.json` in S3 and remove or fix the entry with that `id`.
3. **Invalidate CloudFront cache:**
   - CloudFront > your distribution > **Invalidations > Create invalidation**.
   - Add:
     - `/recordings.json`
     - and, if you removed a file, its path (e.g. `/level1/2025-11-13/video1234567890.mp4`).

After the invalidation completes, the change will be visible in the frontend.

---

## 10. Security and best practices

- Keep `config.json` out of version control.
- Store your `awsAccessKeyId` and `awsSecretAccessKey` in a password manager.
- If a key is compromised, disable the IAM user and create a new one.
- For stricter setups:
  - Make the S3 bucket private.
  - Use a CloudFront Origin Access Control (OAC) so only CloudFront can read the bucket.
  - Limit CORS `AllowedOrigins` to your actual frontend domains.

---

## 11. Known limitations and ideas

- The uploader uses file modification times and a simple “seen file” list (based on filenames) to detect new uploads. If you rename files in `watchDir`, it may re‑upload them.
- No automatic transcoding is performed. Zoom’s MP4 output is uploaded as‑is.
- Transcripts are not generated. You can integrate a separate job (e.g. Whisper, AWS Transcribe) and extend `recordings.json` with subtitle URLs.

---

## 12. Support / contributions

This repository is intended for internal and educational use. If you have suggestions or run into issues:

- Open an issue on the GitHub repository.
- Include:
  - OS (Windows / macOS / Linux),
  - a snippet of any error output from `zoom-recorder`,
  - the HTTP status and body from your `recordings.json` CloudFront URL.

Pull requests are welcome for:

- small UX improvements to the frontend,
- additional storage backends (e.g. S3‑compatible object stores),
- better scheduling / service integration for the uploader.
