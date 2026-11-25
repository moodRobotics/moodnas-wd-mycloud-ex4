<p align="center">
  <img src="assets/brand/mood-robotics.png" alt="Mood Robotics" width="360">
</p>

<h1 align="center">MoodNAS</h1>

<p align="center">
  <strong>Um novo firmware de código aberto que dá uma segunda vida ao WD My Cloud EX4.<br>
  Hardware antigo, segurança moderna, inteligência artificial integrada.</strong>
</p>

<p align="center">
  <a href="README.md">English</a> ·
  <a href="README.es.md">Español</a> ·
  <b>Português</b> ·
  <a href="README.fr.md">Français</a> ·
  <a href="README.de.md">Deutsch</a> ·
  <a href="README.it.md">Italiano</a> ·
  <a href="README.zh-CN.md">简体中文</a> ·
  <a href="README.ja.md">日本語</a>
</p>

<p align="center">
  <img alt="Licença: GPL-3.0" src="https://img.shields.io/badge/licença-GPL--3.0-1f5fff">
  <img alt="Estado: alfa, simulador" src="https://img.shields.io/badge/estado-alfa%20·%20simulador-ffb020">
  <img alt="Idiomas: 8" src="https://img.shields.io/badge/idiomas-8-2fd27a">
</p>

---

## Motivação

O **WD My Cloud EX4** é um NAS sólido de 4 baias que a Western Digital já não
mantém. O seu último firmware usa Linux 3.2 (2013), OpenSSL 1.0.1 e PHP 5.4,
e vem com uma senha de root predefinida. Milhares destes equipamentos ainda
guardam fotos de família e arquivos de pequenas empresas em redes cada vez
mais hostis.

O **MoodNAS** substitui esse firmware por um sistema moderno e aberto.
Aproveita todo o hardware e acrescenta algo que ninguém esperava de uma velha
máquina ARM: **inteligência artificial que protege os seus dados**.

<p align="center">
  <img src="docs/img/dashboard.png" alt="Visão geral do MoodNAS" width="900">
</p>

## Destaques

### 👀 Um NAS com estado de espírito
Os olhos da Mood Robotics são o rosto do MoodNAS:

- **tranquilo** quando está tudo bem;
- **concentrado** quando está ocupado;
- **em repouso** durante a noite;
- **preocupado** quando algo precisa da sua atenção;
- **alarmado**, com um brilho vermelho, quando os seus dados estão em risco.

Os mesmos olhos aparecem no **display OLED frontal** do NAS, para que você
saiba como ele está mesmo do outro lado da sala.

### 🧠 IA que funciona no próprio NAS
Estes modelos não precisam da nuvem e não enviam nada para fora de casa:

| | |
|---|---|
| **Previsão de falhas de disco** | Analisa os dados SMART de cada disco com um modelo baseado nas estatísticas públicas de falhas da Backblaze. Indica qual disco provavelmente vai falhar, e porquê, antes que isso aconteça. |
| **Sentinela anti-ransomware** | Identifica a criptografia em massa de arquivos (análise de entropia e padrões de renomeação) e **congela a pasta compartilhada em modo apenas leitura em poucos segundos**. |
| **Ventilador preditivo** | Antecipa o calor a partir da tendência da temperatura e fica silencioso à noite, sem nunca comprometer a segurança. |
| **Previsão de capacidade** | Aprende a que ritmo você guarda dados e avisa semanas antes de o volume ficar cheio. |

### 💬 "Mood", o seu assistente
Pergunte com as suas próprias palavras: *"Como está o meu NAS?"*, *"Algum disco
é motivo de preocupação?"*, *"Crie uma pasta compartilhada chamada Fotos para
todos"*. O Mood funciona com o **Claude** (API da Anthropic) ou com o
**Ollama na sua própria rede**, sem que nada saia de casa.

- Só consulta o que as suas ferramentas lhe mostram, e qualquer alteração
  aguarda a **sua confirmação**.
- Um **histórico de privacidade** mostra cada pedido que saiu do NAS.

<p align="center">
  <img src="docs/img/ransomware.png" alt="A sentinela anti-ransomware congelou uma pasta compartilhada" width="900">
</p>

### 🔒 Seguro desde o projeto
- **Sem senhas predefinidas.** A configuração inicial exige um código de uso
  único que aparece no display frontal.
- **HTTPS por padrão.**
- **Senhas** com PBKDF2-SHA512, autenticação de dois fatores (TOTP) e
  limitação de tentativas de login.
- **Proteções web:** CSRF, Content-Security-Policy rigorosa e log de
  auditoria.
- **Um único binário estático em Go** com o mínimo de dependências.

Mais detalhes em [SECURITY.md](SECURITY.md).

### 🌍 No seu idioma
A interface e a documentação estão disponíveis em **espanhol, inglês,
português, francês, alemão, italiano, chinês e japonês**. Cada pessoa escolhe
o seu.

### 💾 Os seus dados ficam onde estão
O MoodNAS lê **tal como estão** os volumes RAID criados pelo firmware da WD:
sem formatar e sem migrar.

## Estado

> **O MoodNAS está em fase alfa.** A interface completa, a API, a IA local e
> o assistente já funcionam hoje sobre um **EX4 simulado**. O próximo marco é
> iniciar no hardware real, e você pode ajudar.

| Fase | | Estado |
|---|---|---|
| 1 | Análise do firmware e ferramenta aberta `wdfw` (reempacotamento idêntico byte a byte) | ✅ concluído |
| 2 | Script de inventário do hardware (Stage 0, apenas leitura) | ✅ pronto; faltam relatórios de unidades reais |
| 3 | Interface web (8 idiomas), API, segurança e simulador | ✅ concluído |
| 4 | IA no equipamento e assistente (Claude / Ollama) | ✅ primeira versão |
| 4b | Alertas explicados, pesquisa semântica e etiquetagem de fotos | 🔜 previsto |
| 5 | Kernel *mainline* + Buildroot, inicialização com kexec (sem gravar na flash) e driver do MCU | 🚧 em andamento: *device tree* em rascunho, protocolo do MCU decifrado |
| 6 | Instalação permanente | ⏳ requer uma via de recuperação pelo console serial |

## Experimente em dois minutos

Você precisa do Docker:

```sh
git clone https://github.com/moodrobotics/moodnas-wd-mycloud-ex4.git
cd moodnas-wd-mycloud-ex4
docker compose up --build
```

1. Abra **http://localhost:8080**.
2. Entre com a conta de demonstração **`demo`** / **`moodnas-demo`**.
3. Na página de visão geral, experimente os cenários do simulador: onda de
   calor, disco com falhas, RAID degradado ou ataque de ransomware. Observe os olhos.

## Primeiro acesso e senhas

| Onde | Usuário | Senha |
|---|---|---|
| Demo (`docker compose`, simulador) | `demo` | `moodnas-demo` |
| O seu NAS real | **nenhum por padrão** | **nenhuma por padrão** |

Num NAS real **não existe usuário nem senha por padrão**, de propósito: o
firmware original da WD vinha com uma senha de root conhecida, uma das
fragilidades que o MoodNAS elimina. No primeiro arranque:

1. O NAS mostra um **código de configuração** de 8 caracteres no display
   frontal (e no registro do sistema).
2. Abra `https://<ip-do-nas>` e introduza o código. Isso prova que você está
   fisicamente junto ao NAS, e ninguém mais na sua rede pode apoderar-se dele.
3. Crie **o seu próprio** administrador e senha (mínimo de 10 caracteres).
   Depois pode ativar a autenticação em dois fatores em *A minha conta*.

A conta de demonstração só existe com `moodd -simulate -demo`. O `moodd`
recusa `-demo` em hardware real, porque a senha é pública.

Esqueceu a senha? Procedimento de recuperação: ver
[docs/install](docs/install/README.md) (previsto com o port para o hardware).

## Como funciona

```
Navegador (Svelte, 8 idiomas) ──HTTPS──▶ moodd (Go, um binário estático)
                                         ├─ API · autenticação · auditoria
                                         ├─ IA local: risco de disco · sentinela · ventilador · previsão
                                         ├─ assistente ─▶ API do Claude | Ollama (rede local)
                                         └─ plataforma ─▶ simulador | hardware EX4 (SATA, SMART, MCU, OLED)
```

| | |
|---|---|
| Arquitetura | [docs/architecture](docs/architecture/README.md) |
| A IA em detalhe | [docs/ai](docs/ai/README.md) |
| Hardware | [docs/hardware](docs/hardware/README.md): [formato do firmware](docs/hardware/firmware-format.md) · [protocolo do MCU do painel frontal](docs/hardware/mcu-protocol.md) |
| Plano de instalação e Stage 0 | [docs/install](docs/install/README.md) |

## Hardware

| | WD My Cloud EX4 |
|---|---|
| SoC | Marvell Armada 300 / Kirkwood 88F6282, ARMv5TE, com criptografia por hardware |
| RAM | 512 MB |
| Baias | 4 × SATA de 3,5" |
| Rede | 2 × Gigabit Ethernet |
| Painel frontal | Display OLED, botões e ventilador geridos por um microcontrolador |

Outros modelos da WD (EX2, EX4100, PR4100) ainda não são suportados.
Contribuições são bem-vindas.

## Contribuir

O que mais ajuda neste momento é um
**[relatório Stage 0](docs/install/README.md#stage-0-hardware-inventory)** de um
EX4 real. É apenas de leitura e, além disso, faz um backup da flash.
Consulte [CONTRIBUTING.md](CONTRIBUTING.md) para desenvolvimento e traduções.

## Aviso legal

O MoodNAS é software livre sob a [GNU GPL v3.0 ou posterior](LICENSE). É um
projeto independente da **Mood Robotics** e não é afiliado, endossado nem
suportado pela Western Digital. "WD" e "My Cloud" são marcas comerciais da
Western Digital Corporation, citadas aqui apenas para descrever a
compatibilidade de hardware. Este repositório não contém firmware nem
binários da Western Digital: as ferramentas trabalham sobre arquivos que você
mesmo obtém.

Substituir o firmware de um equipamento tem riscos. O MoodNAS é fornecido sem
garantia: faça primeiro um backup dos seus dados.

<p align="center"><sub>Feito com ♥ pela Mood Robotics: hardware antigo merece inteligência nova.</sub></p>
