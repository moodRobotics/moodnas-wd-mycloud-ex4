<p align="center">
  <img src="assets/brand/mood-robotics.png" alt="Mood Robotics" width="360">
</p>

<h1 align="center">MoodNAS</h1>

<p align="center">
  <strong>Eine neue Open-Source-Firmware, die der WD My Cloud EX4 ein zweites Leben schenkt.<br>
  Alte Hardware, moderne Sicherheit, eingebaute künstliche Intelligenz.</strong>
</p>

<p align="center">
  <a href="README.md">English</a> ·
  <a href="README.es.md">Español</a> ·
  <a href="README.pt.md">Português</a> ·
  <a href="README.fr.md">Français</a> ·
  <b>Deutsch</b> ·
  <a href="README.it.md">Italiano</a> ·
  <a href="README.zh-CN.md">简体中文</a> ·
  <a href="README.ja.md">日本語</a>
</p>

<p align="center">
  <img alt="Lizenz: GPL-3.0" src="https://img.shields.io/badge/Lizenz-GPL--3.0-1f5fff">
  <img alt="Status: Alpha, Simulator" src="https://img.shields.io/badge/Status-Alpha%20·%20Simulator-ffb020">
  <img alt="Sprachen: 8" src="https://img.shields.io/badge/Sprachen-8-2fd27a">
</p>

---

## Warum

Die **WD My Cloud EX4** ist ein solides NAS mit 4 Einschüben, das Western
Digital nicht mehr unterstützt. Ihre letzte Firmware läuft mit Linux 3.2
(2013), OpenSSL 1.0.1 und PHP 5.4 und wird mit einem voreingestellten
Root-Passwort ausgeliefert. Tausende dieser Geräte speichern noch immer
Familienfotos und Dateien kleiner Unternehmen – in Netzwerken, die immer
feindseliger werden.

**MoodNAS** ersetzt diese Firmware durch ein modernes, offenes System. Es
nutzt die gesamte Hardware und ergänzt etwas, das niemand von einer alten
ARM-Kiste erwartet hätte: **künstliche Intelligenz, die deine Daten schützt**.

<p align="center">
  <img src="docs/img/dashboard.png" alt="MoodNAS-Übersicht" width="900">
</p>

## Highlights

### 👀 Ein NAS mit Gemütslage
Die Augen von Mood Robotics sind das Gesicht von MoodNAS:

- **ruhig**, wenn alles in Ordnung ist;
- **konzentriert**, wenn es arbeitet;
- **ruhend** in der Nacht;
- **besorgt**, wenn etwas deine Aufmerksamkeit braucht;
- **alarmiert**, mit rotem Leuchten, wenn deine Daten in Gefahr sind.

Dieselben Augen erscheinen auf dem **OLED-Display an der Front** des NAS – so
siehst du schon von der anderen Seite des Raums, wie es ihm geht.

### 🧠 KI, die direkt auf dem NAS läuft
Diese Modelle brauchen keine Cloud und senden nichts aus deinem Zuhause:

| | |
|---|---|
| **Vorhersage von Festplattenausfällen** | Wertet die SMART-Daten jeder Festplatte mit einem Modell aus, das auf den öffentlichen Ausfallstatistiken von Backblaze basiert. Es sagt dir, welche Festplatte wahrscheinlich ausfallen wird und warum – bevor es passiert. |
| **Ransomware-Wächter** | Erkennt massenhafte Verschlüsselung von Dateien (Entropieanalyse und Umbenennungsmuster) und **friert die Freigabe innerhalb von Sekunden schreibgeschützt ein**. |
| **Vorausschauender Lüfter** | Sieht Hitze anhand des Temperaturtrends voraus und bleibt nachts leise, ohne jemals die Sicherheit zu gefährden. |
| **Kapazitätsprognose** | Lernt, wie schnell du Daten speicherst, und warnt dich Wochen, bevor das Volume voll ist. |

### 💬 „Mood“, dein Assistent
Frag in deinen eigenen Worten: *„Wie geht es meinem NAS?“*, *„Muss ich mir
um eine Festplatte Sorgen machen?“*, *„Leg einen freigegebenen Ordner namens
Fotos für alle an“*. Mood arbeitet mit **Claude** (Anthropic-API) oder mit
**Ollama in deinem eigenen Netzwerk**, sodass nichts dein Zuhause verlässt.

- Es sieht nur, was ihm seine Werkzeuge zeigen, und jede Änderung wartet auf
  **deine Bestätigung**.
- Ein **Datenschutzprotokoll** listet jede Anfrage auf, die das NAS verlassen
  hat.

<p align="center">
  <img src="docs/img/ransomware.png" alt="Der Ransomware-Wächter hat eine Freigabe eingefroren" width="900">
</p>

### 🔒 Sicher von Grund auf
- **Keine Standardpasswörter.** Die Ersteinrichtung verlangt einen
  Einmalcode, der auf dem Front-Display angezeigt wird.
- **HTTPS als Standard.**
- **Passwörter** mit PBKDF2-SHA512 gehasht, Zwei-Faktor-Authentifizierung
  (TOTP) und Drosselung von Anmeldeversuchen.
- **Web-Schutz:** CSRF-Schutz, strikte Content-Security-Policy und
  Audit-Protokoll.
- **Eine einzige statische Go-Binärdatei** mit minimalen Abhängigkeiten.

Mehr dazu in [SECURITY.md](SECURITY.md).

### 🌍 In deiner Sprache
Oberfläche und Dokumentation gibt es auf **Spanisch, Englisch, Portugiesisch,
Französisch, Deutsch, Italienisch, Chinesisch und Japanisch**. Jeder Benutzer
wählt seine eigene Sprache.

### 💾 Deine Daten bleiben, wo sie sind
MoodNAS liest die von der WD-Firmware angelegten RAID-Volumes **so, wie sie
sind**: kein Formatieren, keine Migration.

## Status

> **MoodNAS ist Alpha-Software.** Die komplette Oberfläche, die API, die
> lokale KI und der Assistent laufen heute auf einer **simulierten EX4**. Der
> nächste Meilenstein ist der Start auf der echten Hardware – und du kannst
> dabei helfen.

| Phase | | Stand |
|---|---|---|
| 1 | Firmware-Analyse, offenes Image-Werkzeug `wdfw` (bytegenaues Neuverpacken) | ✅ erledigt |
| 2 | Skript zur Hardware-Inventur (Stage 0, nur lesend) | ✅ bereit; Berichte von echten Geräten fehlen noch |
| 3 | Weboberfläche (8 Sprachen), API, Sicherheit, Simulator | ✅ erledigt |
| 4 | KI auf dem Gerät + Assistent (Claude / Ollama) | ✅ erste Version |
| 4b | Erklärte Warnungen, semantische Suche, Foto-Verschlagwortung | 🔜 geplant |
| 5 | *Mainline*-Kernel + Buildroot, Start per kexec (ohne Schreibzugriff auf den Flash), MCU-Treiber | 🚧 in Arbeit: *Device Tree* als Entwurf, MCU-Protokoll entschlüsselt |
| 6 | Dauerhafte Installation | ⏳ braucht einen Wiederherstellungsweg über die serielle Konsole |

## In zwei Minuten ausprobieren

Du brauchst Docker:

```sh
git clone https://github.com/moodrobotics/moodnas-wd-mycloud-ex4.git
cd moodnas-wd-mycloud-ex4
docker compose up --build
```

1. Öffne **http://localhost:8080**.
2. Melde dich mit dem Demo-Konto **`demo`** / **`moodnas-demo`** an.
3. Probiere auf der Übersichtsseite die Simulator-Szenarien aus: Hitzewelle,
   ausfallende Festplatte, degradiertes RAID, Ransomware-Angriff. Achte auf
   die Augen.

## Erster Zugriff und Passwörter

| Wo | Benutzer | Passwort |
|---|---|---|
| Demo (`docker compose`, Simulator) | `demo` | `moodnas-demo` |
| Dein echtes NAS | **standardmäßig keiner** | **standardmäßig keins** |

Auf einem echten NAS gibt es **keinen Standardbenutzer und kein
Standardpasswort** – mit Absicht: Die Original-Firmware von WD kam mit einem
bekannten Root-Passwort, eine der Schwachstellen, die MoodNAS beseitigt.
Beim ersten Start:

1. Das NAS zeigt auf seinem Frontdisplay (und im Systemprotokoll) einen
   8-stelligen **Einrichtungscode**.
2. Öffne `https://<nas-ip>` und gib den Code ein. Damit beweist du, dass du
   direkt neben dem NAS stehst – niemand sonst in deinem Netz kann es sich
   aneignen.
3. Lege **deinen eigenen** Administrator und ein Passwort an (mindestens 10
   Zeichen). Danach kannst du unter *Mein Konto* die
   Zwei-Faktor-Authentifizierung aktivieren.

Das Demo-Konto existiert nur mit `moodd -simulate -demo`. `moodd` verweigert
`-demo` auf echter Hardware, weil das Passwort öffentlich ist.

Passwort vergessen? Zurücksetzen: siehe [docs/install](docs/install/README.md)
(geplant mit der Hardware-Portierung).

## So funktioniert es

```
Browser (Svelte, 8 Sprachen) ──HTTPS──▶ moodd (Go, eine statische Binärdatei)
                                        ├─ API · Authentifizierung · Audit
                                        ├─ lokale KI: Festplattenrisiko · Wächter · Lüfter · Prognose
                                        ├─ Assistent ─▶ Claude-API | Ollama (LAN)
                                        └─ Plattform ─▶ Simulator | EX4-Hardware (SATA, SMART, MCU, OLED)
```

| | |
|---|---|
| Architektur | [docs/architecture](docs/architecture/README.md) |
| Die KI im Detail | [docs/ai](docs/ai/README.md) |
| Hardware | [docs/hardware](docs/hardware/README.md): [Firmware-Format](docs/hardware/firmware-format.md) · [MCU-Protokoll der Frontblende](docs/hardware/mcu-protocol.md) |
| Installationsplan und Stage 0 | [docs/install](docs/install/README.md) |

## Hardware

| | WD My Cloud EX4 |
|---|---|
| SoC | Marvell Armada 300 / Kirkwood 88F6282, ARMv5TE, mit Hardware-Kryptografie |
| RAM | 512 MB |
| Einschübe | 4 × 3,5" SATA |
| Netzwerk | 2 × Gigabit-Ethernet |
| Frontblende | OLED-Display, Tasten und Lüfter, gesteuert über einen Mikrocontroller |

Andere WD-Modelle (EX2, EX4100, PR4100) werden noch nicht unterstützt.
Beiträge sind willkommen.

## Mitmachen

Am meisten hilft derzeit ein
**[Stage-0-Bericht](docs/install/README.md#stage-0-hardware-inventory)** von
einer echten EX4. Er greift nur lesend zu und sichert außerdem deinen Flash.
In [CONTRIBUTING.md](CONTRIBUTING.md) findest du alles zu Entwicklung und
Übersetzungen.

## Rechtliches

MoodNAS ist freie Software unter der [GNU GPL v3.0 oder neuer](LICENSE). Es
ist ein unabhängiges Projekt von **Mood Robotics** und steht in keiner
Verbindung zu Western Digital, wird von Western Digital weder unterstützt
noch gefördert. „WD“ und „My Cloud“ sind Marken der Western Digital
Corporation und werden hier nur verwendet, um die Hardware-Kompatibilität zu
beschreiben. Dieses Repository enthält keine Firmware und keine
Binärdateien von Western Digital; die Werkzeuge arbeiten mit Dateien, die du
selbst herunterlädst.

Das Ersetzen der Firmware eines Geräts birgt Risiken. MoodNAS wird ohne
Gewährleistung bereitgestellt: Sichere zuerst deine Daten.

<p align="center"><sub>Mit ♥ gemacht von Mood Robotics: Alte Hardware verdient neue Intelligenz.</sub></p>
