<p align="center">
  <img src="assets/brand/mood-robotics.png" alt="Mood Robotics" width="360">
</p>

<h1 align="center">MoodNAS</h1>

<p align="center">
  <strong>Un nuovo firmware open source che dà una seconda vita al WD My Cloud EX4.<br>
  Hardware datato, sicurezza moderna, intelligenza artificiale integrata.</strong>
</p>

<p align="center">
  <a href="README.md">English</a> ·
  <a href="README.es.md">Español</a> ·
  <a href="README.pt.md">Português</a> ·
  <a href="README.fr.md">Français</a> ·
  <a href="README.de.md">Deutsch</a> ·
  <b>Italiano</b> ·
  <a href="README.zh-CN.md">简体中文</a> ·
  <a href="README.ja.md">日本語</a>
</p>

<p align="center">
  <img alt="Licenza: GPL-3.0" src="https://img.shields.io/badge/licenza-GPL--3.0-1f5fff">
  <img alt="Stato: alpha, simulatore" src="https://img.shields.io/badge/stato-alpha%20·%20simulatore-ffb020">
  <img alt="Lingue: 8" src="https://img.shields.io/badge/lingue-8-2fd27a">
</p>

---

## Perché

Il **WD My Cloud EX4** è un solido NAS a 4 bay che Western Digital non
supporta più. Il suo ultimo firmware usa Linux 3.2 (2013), OpenSSL 1.0.1 e
PHP 5.4, e arriva con una password di root predefinita. Migliaia di questi
dispositivi custodiscono ancora foto di famiglia e file di piccole aziende su
reti sempre più ostili.

**MoodNAS** sostituisce quel firmware con un sistema moderno e aperto.
Sfrutta tutto l'hardware e aggiunge qualcosa che nessuno si sarebbe aspettato
da una vecchia macchina ARM: **un'intelligenza artificiale che protegge i tuoi
dati**.

<p align="center">
  <img src="docs/img/dashboard.png" alt="Panoramica di MoodNAS" width="900">
</p>

## In evidenza

### 👀 Un NAS con il suo umore
Gli occhi di Mood Robotics sono il volto di MoodNAS:

- **tranquilli** quando va tutto bene;
- **concentrati** quando è al lavoro;
- **a riposo** di notte;
- **preoccupati** quando qualcosa richiede la tua attenzione;
- **in allarme**, con un bagliore rosso, quando i tuoi dati sono a rischio.

Gli stessi occhi compaiono sul **display OLED frontale**, così puoi capire come
sta anche dall'altra parte della stanza.

### 🧠 IA che gira direttamente sul NAS
Questi modelli non hanno bisogno del cloud e non inviano nulla fuori da casa
tua:

| | |
|---|---|
| **Previsione dei guasti dei dischi** | Analizza i dati SMART di ogni disco con un modello basato sulle statistiche pubbliche sui guasti di Backblaze. Ti dice quale disco probabilmente si guasterà, e perché, prima che succeda. |
| **Sentinella anti-ransomware** | Rileva la cifratura di massa dei file (analisi dell'entropia e schemi di rinomina) e **blocca la cartella condivisa in sola lettura in pochi secondi**. |
| **Ventola predittiva** | Anticipa il calore in base all'andamento della temperatura e resta silenziosa di notte, senza mai compromettere la sicurezza. |
| **Previsione della capacità** | Impara a che ritmo salvi i dati e ti avvisa settimane prima che il volume si riempia. |

### 💬 «Mood», il tuo assistente
Chiedi con parole tue: *«Come sta il mio NAS?»*, *«Devo preoccuparmi per
qualche disco?»*, *«Crea una cartella condivisa chiamata Foto per tutti»*.
Mood funziona con **Claude** (API di Anthropic) oppure con **Ollama sulla tua
rete**, così nulla esce di casa.

- Consulta solo ciò che gli mostrano i suoi strumenti, e ogni modifica attende
  la **tua conferma**.
- Un **registro della privacy** elenca ogni richiesta uscita dal NAS.

<p align="center">
  <img src="docs/img/ransomware.png" alt="La sentinella anti-ransomware ha bloccato una cartella condivisa" width="900">
</p>

### 🔒 Sicuro fin dalla progettazione
- **Nessuna password predefinita.** La configurazione iniziale richiede un
  codice monouso mostrato sul display frontale.
- **HTTPS di default.**
- **Password** con PBKDF2-SHA512, autenticazione a due fattori (TOTP) e
  limitazione dei tentativi di accesso.
- **Protezioni web:** CSRF, una Content-Security-Policy rigorosa e un registro
  di audit.
- **Un unico binario statico in Go** con un insieme minimo di dipendenze.

Maggiori dettagli in [SECURITY.md](SECURITY.md).

### 🌍 Nella tua lingua
Interfaccia e documentazione in **spagnolo, inglese, portoghese, francese,
tedesco, italiano, cinese e giapponese**. Ogni utente sceglie la propria.

### 💾 I tuoi dati restano dove sono
MoodNAS legge **così come sono** i volumi RAID creati dal firmware di WD:
niente formattazione, niente migrazione.

## Stato

> **MoodNAS è in fase alpha.** L'interfaccia completa, l'API, l'IA locale e
> l'assistente funzionano già oggi su un **EX4 simulato**. Il prossimo
> traguardo è l'avvio sull'hardware reale, e puoi dare una mano.

| Fase | | Stato |
|---|---|---|
| 1 | Analisi del firmware e strumento aperto `wdfw` (reimpacchettamento identico byte per byte) | ✅ fatto |
| 2 | Script di inventario dell'hardware (Stage 0, sola lettura) | ✅ pronto; servono report da unità reali |
| 3 | Interfaccia web (8 lingue), API, sicurezza e simulatore | ✅ fatto |
| 4 | IA sul dispositivo e assistente (Claude / Ollama) | ✅ prima versione |
| 4b | Avvisi spiegati, ricerca semantica e tagging delle foto | 🔜 pianificato |
| 5 | Kernel *mainline* + Buildroot, avvio con kexec (senza scritture sulla flash) e driver del MCU | 🚧 in corso: *device tree* in bozza, protocollo del MCU decodificato |
| 6 | Installazione permanente | ⏳ richiede una via di ripristino tramite console seriale |

## Provalo in due minuti

Ti serve Docker:

```sh
git clone https://github.com/moodrobotics/moodnas-wd-mycloud-ex4.git
cd moodnas-wd-mycloud-ex4
docker compose up --build
```

1. Apri **http://localhost:8080**.
2. Accedi con l’account demo **`demo`** / **`moodnas-demo`**.
3. Nella pagina di panoramica, prova gli scenari del simulatore: ondata di
   calore, disco in avaria, RAID degradato, attacco ransomware. Osserva gli
   occhi.

## Primo accesso e password

| Dove | Utente | Password |
|---|---|---|
| Demo (`docker compose`, simulatore) | `demo` | `moodnas-demo` |
| Il tuo NAS reale | **nessuno predefinito** | **nessuna predefinita** |

Su un NAS reale **non esistono utente né password predefiniti**, di
proposito: il firmware originale di WD aveva una password di root nota, uno
dei punti deboli che MoodNAS elimina. Al primo avvio:

1. Il NAS mostra un **codice di configurazione** di 8 caratteri sul display
   frontale (e nel log di sistema).
2. Apri `https://<ip-del-nas>` e inserisci il codice. Così dimostri di essere
   fisicamente accanto al NAS e nessun altro nella tua rete può
   appropriarsene.
3. Crea **il tuo** amministratore e la tua password (almeno 10 caratteri).
   Poi puoi attivare l’autenticazione a due fattori in *Il mio account*.

L’account demo esiste solo con `moodd -simulate -demo`. `moodd` rifiuta
`-demo` su hardware reale, perché la password è pubblica.

Password dimenticata? Procedura di ripristino: vedi
[docs/install](docs/install/README.md) (prevista con il porting hardware).

## Come funziona

```
Browser (Svelte, 8 lingue) ──HTTPS──▶ moodd (Go, un binario statico)
                                      ├─ API · autenticazione · audit
                                      ├─ IA locale: rischio dischi · sentinella · ventola · previsione
                                      ├─ assistente ─▶ API di Claude | Ollama (LAN)
                                      └─ piattaforma ─▶ simulatore | hardware EX4 (SATA, SMART, MCU, OLED)
```

| | |
|---|---|
| Architettura | [docs/architecture](docs/architecture/README.md) |
| L'IA nel dettaglio | [docs/ai](docs/ai/README.md) |
| Hardware | [docs/hardware](docs/hardware/README.md): [formato del firmware](docs/hardware/firmware-format.md) · [protocollo del MCU del pannello frontale](docs/hardware/mcu-protocol.md) |
| Piano di installazione e Stage 0 | [docs/install](docs/install/README.md) |

## Hardware

| | WD My Cloud EX4 |
|---|---|
| SoC | Marvell Armada 300 / Kirkwood 88F6282, ARMv5TE, con crittografia hardware |
| RAM | 512 MB |
| Bay | 4 × SATA da 3,5" |
| Rete | 2 × Gigabit Ethernet |
| Pannello frontale | Display OLED, pulsanti e ventola gestiti da un microcontrollore |

Gli altri modelli WD (EX2, EX4100, PR4100) non sono ancora supportati. I
contributi sono benvenuti.

## Contribuire

L'aiuto più utile in questo momento è un
**[report Stage 0](docs/install/README.md#stage-0-hardware-inventory)** da un
EX4 reale. È in sola lettura e fa anche un backup della flash. Consulta
[CONTRIBUTING.md](CONTRIBUTING.md) per sviluppo e traduzioni.

## Note legali

MoodNAS è software libero distribuito sotto [GNU GPL v3.0 o successiva](LICENSE).
È un progetto indipendente di **Mood Robotics** e non è affiliato, approvato
né supportato da Western Digital. «WD» e «My Cloud» sono marchi di Western
Digital Corporation, citati qui solo per descrivere la compatibilità
hardware. Questo repository non contiene firmware né binari di Western
Digital: gli strumenti lavorano sui file che scarichi tu.

Sostituire il firmware di un dispositivo comporta dei rischi. MoodNAS è
fornito senza alcuna garanzia: fai prima un backup dei tuoi dati.

<p align="center"><sub>Fatto con ♥ da Mood Robotics: l'hardware datato merita un'intelligenza nuova.</sub></p>
