<p align="center">
  <img src="assets/brand/mood-robotics.png" alt="Mood Robotics" width="360">
</p>

<h1 align="center">MoodNAS</h1>

<p align="center">
  <strong>Un nouveau firmware open source qui offre une seconde vie au WD My Cloud EX4.<br>
  Matériel ancien, sécurité moderne, intelligence artificielle intégrée.</strong>
</p>

<p align="center">
  <a href="README.md">English</a> ·
  <a href="README.es.md">Español</a> ·
  <a href="README.pt.md">Português</a> ·
  <b>Français</b> ·
  <a href="README.de.md">Deutsch</a> ·
  <a href="README.it.md">Italiano</a> ·
  <a href="README.zh-CN.md">简体中文</a> ·
  <a href="README.ja.md">日本語</a>
</p>

<p align="center">
  <img alt="Licence : GPL-3.0" src="https://img.shields.io/badge/licence-GPL--3.0-1f5fff">
  <img alt="Statut : alpha, simulateur" src="https://img.shields.io/badge/statut-alpha%20·%20simulateur-ffb020">
  <img alt="Langues : 8" src="https://img.shields.io/badge/langues-8-2fd27a">
</p>

---

## Pourquoi

Le **WD My Cloud EX4** est un NAS solide à 4 baies que Western Digital ne
prend plus en charge. Son dernier firmware tourne sous Linux 3.2 (2013),
OpenSSL 1.0.1 et PHP 5.4, et il est livré avec un mot de passe root par
défaut. Des milliers de ces boîtiers conservent encore des photos de famille
et des fichiers de petites entreprises sur des réseaux de plus en plus
hostiles.

**MoodNAS** remplace ce firmware par un système moderne et ouvert. Il exploite
tout le matériel et lui ajoute ce que personne n'attendait d'un vieux boîtier
ARM : **une intelligence artificielle qui protège vos données**.

<p align="center">
  <img src="docs/img/dashboard.png" alt="Vue d'ensemble de MoodNAS" width="900">
</p>

## Points forts

### 👀 Un NAS qui a de l'humeur
Les yeux de Mood Robotics sont le visage de MoodNAS :

- **serein** quand tout va bien ;
- **concentré** quand il travaille ;
- **au repos** la nuit ;
- **inquiet** quand quelque chose demande votre attention ;
- **alarmé**, avec un halo rouge, quand vos données sont en danger.

Les mêmes yeux s'affichent sur l'**écran OLED en façade** du NAS : vous savez
comment il va depuis l'autre bout de la pièce.

### 🧠 Une IA qui tourne sur le NAS lui-même
Ces modèles n'ont pas besoin du cloud et n'envoient rien hors de chez vous :

| | |
|---|---|
| **Prédiction des pannes de disque** | Analyse les données SMART de chaque disque avec un modèle fondé sur les statistiques de pannes publiées par Backblaze. Il vous indique quel disque risque de lâcher, et pourquoi, avant que cela n'arrive. |
| **Sentinelle anti-ransomware** | Repère le chiffrement massif de fichiers (analyse d'entropie et schémas de renommage) et **gèle le partage en lecture seule en quelques secondes**. |
| **Ventilateur prédictif** | Anticipe la chaleur d'après la tendance des températures et reste discret la nuit, sans jamais compromettre la sécurité. |
| **Prévision de capacité** | Apprend à quel rythme vous stockez des données et vous prévient des semaines avant que le volume ne soit plein. |

### 💬 « Mood », votre assistant
Posez vos questions avec vos propres mots : *« Comment va mon NAS ? »*,
*« Dois-je m'inquiéter pour un disque ? »*, *« Crée un dossier partagé
appelé Photos pour tout le monde »*. Mood fonctionne avec **Claude** (API
d'Anthropic) ou avec **Ollama sur votre propre réseau**, sans que rien ne
sorte de chez vous.

- Il ne consulte que ce que ses outils lui montrent, et chaque modification
  attend **votre confirmation**.
- Un **journal de confidentialité** répertorie chaque requête sortie du NAS.

<p align="center">
  <img src="docs/img/ransomware.png" alt="La sentinelle anti-ransomware a gelé un partage" width="900">
</p>

### 🔒 Sécurisé dès la conception
- **Aucun mot de passe par défaut.** La configuration initiale exige un code
  à usage unique affiché sur l'écran en façade.
- **HTTPS par défaut.**
- **Mots de passe** hachés en PBKDF2-SHA512, authentification à deux facteurs
  (TOTP) et limitation des tentatives de connexion.
- **Protections web :** CSRF, Content-Security-Policy stricte et journal
  d'audit.
- **Un seul binaire Go statique** avec un minimum de dépendances.

Plus de détails dans [SECURITY.md](SECURITY.md).

### 🌍 Dans votre langue
L'interface et la documentation sont disponibles en **espagnol, anglais,
portugais, français, allemand, italien, chinois et japonais**. Chaque
utilisateur choisit la sienne.

### 💾 Vos données restent où elles sont
MoodNAS lit **tels quels** les volumes RAID créés par le firmware de WD : ni
formatage, ni migration.

## Statut

> **MoodNAS est un logiciel en phase alpha.** L'interface complète, l'API,
> l'IA locale et l'assistant fonctionnent aujourd'hui sur un **EX4 simulé**.
> La prochaine étape est le démarrage sur le vrai matériel, et vous pouvez
> nous aider.

| Phase | | État |
|---|---|---|
| 1 | Analyse du firmware et outil ouvert `wdfw` (réempaquetage identique à l'octet près) | ✅ terminé |
| 2 | Script d'inventaire matériel (Stage 0, lecture seule) | ✅ prêt ; il manque des rapports d'appareils réels |
| 3 | Interface web (8 langues), API, sécurité et simulateur | ✅ terminé |
| 4 | IA embarquée et assistant (Claude / Ollama) | ✅ première version |
| 4b | Alertes expliquées, recherche sémantique et étiquetage des photos | 🔜 prévu |
| 5 | Noyau *mainline* + Buildroot, démarrage via kexec (sans écriture en flash) et pilote du MCU | 🚧 en cours : *device tree* à l'état de brouillon, protocole du MCU décodé |
| 6 | Installation permanente | ⏳ nécessite une procédure de récupération par console série |

## Essayez-le en deux minutes

Il vous faut Docker :

```sh
git clone https://github.com/moodrobotics/moodnas-wd-mycloud-ex4.git
cd moodnas-wd-mycloud-ex4
docker compose up --build
```

1. Ouvrez **http://localhost:8080**.
2. Connectez-vous avec le compte de démonstration **`demo`** / **`moodnas-demo`**.
3. Sur la page de vue d'ensemble, essayez les scénarios du simulateur : vague
   de chaleur, disque défaillant, RAID dégradé ou attaque par ransomware.
   Observez les yeux.

## Premier accès et mots de passe

| Où | Utilisateur | Mot de passe |
|---|---|---|
| Démo (`docker compose`, simulateur) | `demo` | `moodnas-demo` |
| Votre vrai NAS | **aucun par défaut** | **aucun par défaut** |

Sur un vrai NAS, il n’y a **ni utilisateur ni mot de passe par défaut**, et
c’est voulu : le firmware d’origine de WD était livré avec un mot de passe
root connu, l’une des faiblesses que MoodNAS supprime. Au premier démarrage :

1. Le NAS affiche un **code de configuration** de 8 caractères sur son écran
   frontal (et dans le journal système).
2. Ouvrez `https://<ip-du-nas>` et saisissez le code. Cela prouve que vous
   êtes physiquement à côté du NAS : personne d’autre sur votre réseau ne
   peut se l’approprier.
3. Créez **votre propre** administrateur et mot de passe (10 caractères
   minimum). Vous pourrez ensuite activer la double authentification dans
   *Mon compte*.

Le compte de démonstration n’existe qu’avec `moodd -simulate -demo`. `moodd`
refuse `-demo` sur du vrai matériel, car son mot de passe est public.

Mot de passe oublié ? Procédure de réinitialisation : voir
[docs/install](docs/install/README.md) (prévue avec le portage matériel).

## Comment ça marche

```
Navigateur (Svelte, 8 langues) ──HTTPS──▶ moodd (Go, un binaire statique)
                                          ├─ API · authentification · audit
                                          ├─ IA locale : risque disque · sentinelle · ventilateur · prévision
                                          ├─ assistant ─▶ API Claude | Ollama (réseau local)
                                          └─ plateforme ─▶ simulateur | matériel EX4 (SATA, SMART, MCU, OLED)
```

| | |
|---|---|
| Architecture | [docs/architecture](docs/architecture/README.md) |
| L'IA en détail | [docs/ai](docs/ai/README.md) |
| Matériel | [docs/hardware](docs/hardware/README.md) : [format du firmware](docs/hardware/firmware-format.md) · [protocole du MCU de la façade](docs/hardware/mcu-protocol.md) |
| Plan d'installation et Stage 0 | [docs/install](docs/install/README.md) |

## Matériel

| | WD My Cloud EX4 |
|---|---|
| SoC | Marvell Armada 300 / Kirkwood 88F6282, ARMv5TE, avec chiffrement matériel |
| RAM | 512 Mo |
| Baies | 4 × SATA 3,5" |
| Réseau | 2 × Gigabit Ethernet |
| Façade | Écran OLED, boutons et ventilateur gérés par un microcontrôleur |

Les autres modèles WD (EX2, EX4100, PR4100) ne sont pas encore pris en charge.
Les contributions sont les bienvenues.

## Contribuer

L'aide la plus utile en ce moment est un
**[rapport Stage 0](docs/install/README.md#stage-0-hardware-inventory)** issu
d'un vrai EX4. Il fonctionne en lecture seule et sauvegarde aussi votre
flash. Consultez [CONTRIBUTING.md](CONTRIBUTING.md) pour le développement et
les traductions.

## Mentions légales

MoodNAS est un logiciel libre distribué sous [GNU GPL v3.0 ou ultérieure](LICENSE).
C'est un projet indépendant de **Mood Robotics**, qui n'est ni affilié à
Western Digital, ni approuvé ou soutenu par cette société. « WD » et
« My Cloud » sont des marques de Western Digital Corporation, citées ici
uniquement pour décrire la compatibilité matérielle. Ce dépôt ne contient
aucun firmware ni aucun binaire de Western Digital : les outils travaillent
sur des fichiers que vous téléchargez vous-même.

Remplacer le firmware d'un appareil comporte des risques. MoodNAS est fourni
sans garantie : sauvegardez d'abord vos données.

<p align="center"><sub>Fait avec ♥ par Mood Robotics : le matériel ancien mérite une intelligence nouvelle.</sub></p>
