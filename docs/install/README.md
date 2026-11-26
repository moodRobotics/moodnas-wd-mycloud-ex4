# Installing MoodNAS

> **Status:** MoodNAS runs on a real EX4 today as a **sidecar** next to WD's
> firmware: it observes disks, SMART, RAID and network, with no installation
> and no flash writes. Replacing WD's firmware is phase 5.

## Safety principles

1. **Your data first.** MoodNAS reads the RAID volumes created by WD's firmware
   as they are (mdadm + ext4). Nothing is formatted or migrated.
2. **No flash writes until a recovery path exists.** Without a serial
   console, a bad NAND write can leave the NAS unbootable. MoodNAS therefore
   starts from the factory firmware with **kexec** and runs from a disk or USB
   partition. Turning the NAS off and on brings back the original firmware.
3. **Permanent installation needs a USB-TTL serial adapter** (about 3 €).
   With it, a Kirkwood SoC can be recovered over UART (`kwboot`) even if the
   flash is erased.

## Stage 0: hardware inventory

This is the most useful thing you can do today. It is read-only.

1. In the WD web interface, enable SSH: *Settings → Network → SSH*.
2. Copy [tools/stage0/inventory.sh](../../tools/stage0/inventory.sh) to a share
   (for example `Public`).
3. Connect with `ssh sshd@<nas-ip>` and run:

   ```sh
   sh /shares/Public/inventory.sh /shares/Public
   ```

4. You get a `moodnas-stage0-<date>/` folder:
   - `report-<date>.txt`: please attach it to a GitHub issue. It confirms the
     flash layout, RAM size, kexec support, bay mapping and MCU protocol.
   - `mtd/`: a complete backup of the NAND flash. **Keep it private**: it
     contains your MAC addresses and serial number. With a serial console it
     can restore the NAS exactly as it left the factory.

## Run MoodNAS on your EX4 today (sidecar)

1. Build the package on a computer (`make sidecar`) or download it from the
   releases, and copy the `moodnas-sidecar` folder into the NAS's `Public`
   share.
2. Over SSH on the NAS:

   ```sh
   sh /shares/Public/moodnas-sidecar/moodnas-sidecar.sh start
   ```

3. Open `https://<nas-ip>:8443`, accept the self-signed certificate, and
   enter the setup code that the script prints.

`stop`, `status` and `log` work the same way. To remove it, run `stop` and
delete the folder. In this mode WD's firmware keeps control of the fan, the
display and the buttons; MoodNAS only reads. It runs at the lowest priority
and reads SMART at most every 10 minutes, so it is safe during a RAID rebuild.

## Phase 5 boot plan (in development)

```
power on → U-Boot → WD kernel 3.2 + ramdisk (unchanged, from NAND)
         → moodnas-boot.sh (started from a share or USB)
         → kexec: mainline kernel + MoodNAS rootfs from a disk/USB partition
         → MoodNAS runs; a power cycle returns to WD firmware
```

Stage 0 confirmed that WD's kernel has **no kexec support**, so the next
step is a kexec loadable module for 3.2.40 built from WD's own GPL sources
(same compiler and configuration, so the module loads on the stock kernel).

## Try it now: simulator

```sh
git clone https://github.com/moodrobotics/moodnas-wd-mycloud-ex4 && cd moodnas-wd-mycloud-ex4
docker compose up --build
```

Open <http://localhost:8080>, sign in with the public demo account
`demo` / `moodnas-demo` (it exists only in the simulator), and play with the
scenarios on the overview page (heat wave, failing disk, degraded RAID,
ransomware attack…).

On a real NAS there is no default account. The first administrator is
created through the one-time setup code shown on the front display.
