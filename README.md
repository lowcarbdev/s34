# s34

A command-line tool for the [ARRIS SURFboard S34](https://www.arris.com) cable modem. Interact with the modem's HNAP JSON API to check signal levels, read the event log, and restart the modem without touching the web UI.

## Installation

```sh
go install github.com/lowcarbdev/s34@latest
```

Or build from source:

```sh
git clone https://github.com/lowcarbdev/s34
cd s34
go build
```

Requires Go 1.26+.

## Usage

```
s34 [command] [flags]
```

The password can be supplied via flag or environment variable:

```sh
export S34_PASSWORD='your-password'
s34 info
```

### Commands

| Command | Description |
|---|---|
| `info` | Device identity — model, MAC, serial, firmware, uptime |
| `channels` | Downstream and upstream channel bonding summary |
| `status connection` | Connection status and network access |
| `status downstream` | Downstream channel detail |
| `status upstream` | Upstream channel detail |
| `status startup` | Startup sequence / boot status |
| `status software` | Firmware and software versions |
| `log` | Event log, one entry per line |
| `watch <command>` | Refresh any command on a loop |
| `restart` | Reboot the modem |
| `hnap <Action>` | Send a raw HNAP action (for exploration) |

### Global flags

```
-p, --password string   Admin password (env: S34_PASSWORD)
-u, --username string   Admin username (env: S34_USERNAME, default: admin)
    --url string        Modem base URL (default "https://192.168.100.1")
    --json              Output raw JSON
```

## Examples

Device summary:

```
$ s34 info
Model:       S34
Serial:      xxxxxxxxxxxxxxxxxxx
MAC:         xx:xx:xx:xx:xx:xx
Firmware:    AT01.01.007.032024_S3.04.735
Hardware:    1.0
Internet:    Connected
Network:     Allowed
Uptime:      0 days 10h:40m:43s
System time: 04/14/2026 09:14:50
```

Channel bonding:

```
$ s34 channels
Downstream (34 channels)
------------------------------------------------------------------------------------------
Ch    Status    Modulation  ID    Freq (Hz)     Pwr (dBmV)  SNR (dB)  Corrected   Uncorrected
------------------------------------------------------------------------------------------
1     Locked    256QAM      1     459000000     3.4         40.9      1           0
2     Locked    256QAM      5     483000000     3.1         40.9      0           0
...
33    Locked    OFDM PLC    193   261000000     5.8         44.0      81894739    2
34    Locked    OFDM PLC    194   669000000     0.9         42.0      120550290   4

Upstream (4 channels)
------------------------------------------------------------------------
Ch    Status    Type      ID    Sym Rate (Ksym)  Freq (Hz)     Pwr (dBmV)
------------------------------------------------------------------------
1     Locked    SC-QAM    3     6400000          36000000      42.0
2     Locked    SC-QAM    1     6400000          23200000      42.8
3     Locked    SC-QAM    2     6400000          29600000      42.0
4     Locked    OFDMA     41    10000000         5700000       37.2
```

Event log:

```
$ s34 log
04/14/2026 09:14:51  Notice      Successful LAN WebGUI login from 192.168.100.2
04/14/2026 07:36:50  Notice      US profile assignment change. US Chan ID: 41; Previous Profile: 9; New Profile: 6.
04/13/2026 18:45:01  Critical    No Ranging Response received - T3 time-out
04/13/2026 18:44:56  Critical    SYNC Timing Synchronization failure - Loss of Sync
```

Watch channel levels refresh every 10 seconds:

```
$ s34 watch --interval 10s channels
```

Restart the modem:

```
$ s34 restart
Logging in as admin...
Sending restart command...
Modem is restarting. It will be unreachable for approximately 2 minutes.
```

Raw HNAP exploration:

```
$ s34 hnap GetCustomerStatusSoftware
```

## Scheduled restarts with systemd

You can use a systemd service and timer to restart the modem on a schedule. The example below runs at 10:30 AM every Wednesday and Sunday. The time will match your system time. If in UTC, adjust the offset accordingly.

### 1. Store the password

Create a credentials file readable only by root:

```sh
sudo mkdir -p /etc/s34
sudo sh -c "echo 'S34_PASSWORD=your-password' > /etc/s34/credentials"
sudo chmod 600 /etc/s34/credentials
```

### 2. Create the service unit

`/etc/systemd/system/s34-restart.service`:

```ini
[Unit]
Description=Restart ARRIS SURFboard S34 modem
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
EnvironmentFile=/etc/s34/credentials
ExecStart=/usr/local/bin/s34 restart
```

### 3. Create the timer unit

`/etc/systemd/system/s34-restart.timer`:

```ini
[Unit]
Description=Restart S34 modem weekly on Wednesday and Sunday at 3:30 AM system time

[Timer]
OnCalendar=Wed,Sun 10:30:00
Persistent=true

[Install]
WantedBy=timers.target
```

### 4. Install and enable

Copy the binary, reload systemd, and start the timer:

```sh
sudo cp s34 /usr/local/bin/s34
sudo systemctl daemon-reload
sudo systemctl enable s34-restart.timer
sudo systemctl start s34-restart.timer
```

Verify the timer is active and check when it will next fire:

```sh
systemctl status s34-restart.timer
systemctl list-timers s34-restart.timer
```

Check logs from past runs:

```sh
journalctl -u s34-restart.service
```

## Notes

- The modem uses a self-signed TLS certificate issued by an internal ARRIS CA. TLS verification is intentionally skipped.
- The modem locks out login attempts after a small number of failures. If you see `account locked`, wait a few minutes before retrying.
- During startup after a restart, the modem returns `RELOAD` before it is ready to accept commands. The `restart` command retries automatically for up to 5 minutes.
- The downstream channel endpoint has a firmware bug where channel data is injected into an HTTP response header, causing standard HTTP clients to reject the response. This tool works around it by using a raw TLS connection for that endpoint.

## License

MIT
