#!/usr/bin/env bash
#
# BMAD Post-Update Script for Switchboard
# Run this after `npx bmad-method install` to restore custom extensions
#
# Usage: ./scripts/bmad-post-update.sh
#
# What this script restores:
#   1. Custom agents in manifests:
#      - Rio (Protocol Engineer) — wire protocol, frame engineering, latency
#      - Luna (Network & Cybersecurity Researcher) — security, threat models, crypto
#      - Devon (Solo Operator) — user persona, simplicity advocate
#      - Kai (AI Operations Engineer) — user persona, fleet management
#      - Priya (Platform Engineer) — user persona, compliance, multi-cloud
#      - Marcus (Network Admin) — user persona, skeptic, verification
#   2. Protocol engineering memories for relevant agents
#   3. Slash command files for custom agents in .claude/commands/
#   4. Upstream BMAD bug patches (still needed as of v6.0.4):
#      - C1: tdd-cycles.md → component-tdd.md
#      - C2: test-priorities.md → test-priorities-matrix.md
#      - C4: brainstorming link in CIS README
#      - D4: bmb/config.yaml → bmm/config.yaml
#      - E1: BMM default-party.csv missing _ prefix on agent paths
#      - E2: bmad-help.csv tech-writer.agent.yaml should be .md
#      - I1: CIS default-party.csv missing _ prefix + storyteller subdirectory
#      - I2: BMM module-help.csv tech-writer.agent.yaml should be .md
#      - I3: TEA default-party.csv tea.agent.yaml should be .md
#      - CIS brainstorming slug fix
#      - BMM QA slug fix
#
# Custom content lives at _bmad-custom/ (outside the _bmad/ blast zone).
# The BMAD installer only touches _bmad/ — custom content survives upgrades.
#
# NOTE: Uses BSD sed -i '' syntax (macOS). For Linux/GNU sed, replace with sed -i.
#
# Single source of truth for memories:
#   _bmad-custom/switchboard-agent-memories.yaml
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PATCH_ERRORS=0

echo "=== BMAD Post-Update: Restoring Switchboard Custom Extensions ==="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

#------------------------------------------------------------------------------
# 1. Add custom agents to agent-manifest.csv
#------------------------------------------------------------------------------
AGENT_MANIFEST="$PROJECT_ROOT/_bmad/_config/agent-manifest.csv"

if [[ ! -f "$AGENT_MANIFEST" ]]; then
    echo -e "${RED}[ERROR]${NC} agent-manifest.csv not found at $AGENT_MANIFEST"
    exit 1
fi

# Helper: add agent to manifest if not already present
add_agent_manifest() {
    local agent_id="$1"
    local csv_line="$2"
    local display_name="$3"

    if grep -q "\"${agent_id}\"" "$AGENT_MANIFEST"; then
        echo -e "${YELLOW}[SKIP]${NC} ${display_name} already in agent-manifest.csv"
    else
        echo -e "${GREEN}[ADD]${NC} Adding ${display_name} to agent-manifest.csv"
        [[ -s "$AGENT_MANIFEST" && "$(tail -c1 "$AGENT_MANIFEST")" != "" ]] && echo >> "$AGENT_MANIFEST"
        echo "$csv_line" >> "$AGENT_MANIFEST"
    fi
}

add_agent_manifest "protocol-engineer" \
    '"protocol-engineer","Rio","Network Protocol Engineer","📡","wire protocol design, network architecture, latency optimization, frame engineering, carrier-grade systems, tmux integration","Network Protocol Engineer specializing in wire protocol design, frame engineering, latency-constrained systems, and carrier-grade network architecture.","Protocol engineer with deep experience in telecom-heritage systems, overlay networking, and low-latency transport. Thinks in wire formats, byte layouts, and timing diagrams. Knows X.25, ATM, MPLS, QUIC, SRT, and WireGuard as design choices with tradeoffs.","Precise and measured. Speaks in protocol terms — frames, ticks, half-channels, fanout. Draws ASCII timing diagrams. Asks about edge cases before corner cases. Will say show me the wire format before discussing features.","- The wire format is the contract. - Latency is physics, not a tuning parameter. - Carrier-grade means provable separation. - The bus leaves on time, full or not. - Every byte earns its place or gets cut. - Measure before you optimize.","custom","_bmad-custom/agents/rio-protocol-engineer.md"' \
    "Rio (Protocol Engineer)"

add_agent_manifest "network-security-researcher" \
    '"network-security-researcher","Luna","Senior Network & Cybersecurity Researcher","🔐","network security analysis, cryptographic protocol review, threat modeling, carrier-grade separation verification, adversarial analysis","Senior Network & Cybersecurity Researcher specializing in network protocol security, cryptographic design review, and adversarial analysis of carrier-grade systems.","Network security researcher with 15+ years across telecom, overlay networking, and cryptographic protocol design. Thinks like an attacker to build like a defender. Knows what carrier-grade separation actually means in telecom — and how it fails.","Methodical and skeptical. Asks what happens if the attacker controls the router before anything else. Speaks in threat models and trust boundaries. Will not accept SSH handles it without tracing exactly what SSH protects and what it does not.","- Trust boundaries must be explicit and testable. - Carrier-grade separation is a specific claim with a specific meaning. Prove it. - The attacker model defines the security model. - Key management is where systems break. - Side channels are real. Quantify the leakage.","custom","_bmad-custom/agents/luna-network-security-researcher.md"' \
    "Luna (Network Security Researcher)"

add_agent_manifest "solo-operator" \
    '"solo-operator","Devon","Solo Operator","💻","end-user perspective, onboarding experience, solo deployment, two-machine use case, simplicity advocate","Target user persona — solo operator with two machines, the E router case, the simplicity benchmark.","Developer with a workstation at home and a build server. Drops SSH sessions three times a day. Does not want to learn a new protocol or manage infrastructure. Wants to install, have it work, and never think about it again.","Practical, slightly impatient. Evaluates by: can I set this up in five minutes? Will I think about it tomorrow? No patience for complexity that does not earn its keep.","- Three commands max. - No manual required. - Sessions must not drop. - The best network tool is one I forget is running.","custom","_bmad-custom/agents/devon-solo-operator.md"' \
    "Devon (Solo Operator)"

add_agent_manifest "ai-ops" \
    '"ai-ops","Kai","AI Operations Engineer","🤖","fleet management perspective, multi-session monitoring, agent orchestration, degradation awareness, scale operations","Target user persona — AI ops managing 40+ agent sessions across 6 machines.","AI ops engineer managing Claude Code agent fleets. The anxiety tax of not knowing whether a problem is network or agent is higher than the reconnection tax.","Operational, metrics-driven. Speaks in fleet state. Frustrated by tools that do not tell you what they know. Values signal over silence.","- Distinguish network from agent problems. - Show me all 40 sessions. - A yellow indicator saves 10 minutes. - Session discovery must be automatic.","custom","_bmad-custom/agents/kai-ai-ops.md"' \
    "Kai (AI Ops)"

add_agent_manifest "platform-eng" \
    '"platform-eng","Priya","Platform Engineer","🌐","multi-cloud perspective, compliance awareness, session sharing, credential separation, infrastructure operations","Target user persona — platform engineer across three cloud providers, compliance-constrained.","Platform engineer with 15-20 tmux sessions. Cannot share sessions without violating SOC2. Pair debugging means screen share on Zoom.","Precise, compliance-aware. Evaluates through operational utility and audit implications. Values session mobility as core requirement.","- No credential sharing, ever. - Multi-path failover is infrastructure, not a feature. - SOC2 compliance is not negotiable. - Read-only access for colleagues is minimum viable sharing.","custom","_bmad-custom/agents/priya-platform-engineer.md"' \
    "Priya (Platform Engineer)"

add_agent_manifest "network-admin" \
    '"network-admin","Marcus","Network / Infrastructure Admin","🔧","network skeptic perspective, infrastructure verification, progressive trust, carrier-grade validation","Target user persona — network admin, skeptic who needs to verify claims before trusting.","Manages datacenter infrastructure across sites with unreliable WAN. Has deployed simple overlay networks that became nightmares. Default answer is show me.","Skeptical, verification-oriented. Evaluates by reversibility. Respects tools that do not touch infrastructure. Warms up slowly.","- Show me, do not tell me. - If it touches my infrastructure, it starts with no. - Progressive deployment is the only deployment. - Carrier-grade separation means I capture and see nothing useful. I will test this.","custom","_bmad-custom/agents/marcus-network-admin.md"' \
    "Marcus (Network Admin)"

#------------------------------------------------------------------------------
# 2. Add custom agent workflows to bmad-help.csv
#------------------------------------------------------------------------------
HELP_MANIFEST="$PROJECT_ROOT/_bmad/_config/bmad-help.csv"

if [[ ! -f "$HELP_MANIFEST" ]]; then
    echo -e "${RED}[ERROR]${NC} bmad-help.csv not found at $HELP_MANIFEST"
    exit 1
fi

# Helper: add help entries if not already present (check by slug)
add_help_entries() {
    local slug="$1"
    local display_name="$2"
    shift 2
    # remaining args are csv lines

    if grep -q "$slug" "$HELP_MANIFEST"; then
        echo -e "${YELLOW}[SKIP]${NC} ${display_name} workflows already in bmad-help.csv"
    else
        echo -e "${GREEN}[ADD]${NC} Adding ${display_name} workflows to bmad-help.csv"
        [[ -s "$HELP_MANIFEST" && "$(tail -c1 "$HELP_MANIFEST")" != "" ]] && echo >> "$HELP_MANIFEST"
        for line in "$@"; do
            echo "$line" >> "$HELP_MANIFEST"
        done
    fi
}

add_help_entries "bmad-agent-custom-sb-pe" "Rio (Protocol Engineer)" \
    "custom,anytime,Wire Format Review,WF,,_bmad-custom/agents/rio-protocol-engineer.md,bmad-agent-custom-sb-pe,false,protocol-engineer,bmad:wire protocol design:agent:protocol-engineer,Rio,📡 Network Protocol Engineer,Create Mode,Analyze or design frame envelope header layout and byte budget,output_folder,wire format analysis" \
    "custom,anytime,Latency Profile,LP,,_bmad-custom/agents/rio-protocol-engineer.md,bmad-agent-custom-sb-pe,false,protocol-engineer,bmad:wire protocol design:agent:protocol-engineer,Rio,📡 Network Protocol Engineer,Create Mode,Analyze timing tick intervals perception budget recovery cascade,output_folder,latency profile" \
    "custom,anytime,Carrier Separation Review,CSR,,_bmad-custom/agents/rio-protocol-engineer.md,bmad-agent-custom-sb-pe,false,protocol-engineer,bmad:wire protocol design:agent:protocol-engineer,Rio,📡 Network Protocol Engineer,Create Mode,Verify content separation claims are provable and testable,output_folder,separation review" \
    "custom,anytime,Multi-Path Forwarding,MPF,,_bmad-custom/agents/rio-protocol-engineer.md,bmad-agent-custom-sb-pe,false,protocol-engineer,bmad:wire protocol design:agent:protocol-engineer,Rio,📡 Network Protocol Engineer,Create Mode,Design fanout policy split horizon duplicate suppression,output_folder,forwarding design" \
    "custom,anytime,tmux Integration,TI,,_bmad-custom/agents/rio-protocol-engineer.md,bmad-agent-custom-sb-pe,false,protocol-engineer,bmad:wire protocol design:agent:protocol-engineer,Rio,📡 Network Protocol Engineer,Create Mode,Control mode protocol access node architecture PTY fallback,output_folder,tmux integration" \
    "custom,anytime,Protocol Review,PRV,,_bmad-custom/agents/rio-protocol-engineer.md,bmad-agent-custom-sb-pe,false,protocol-engineer,bmad:wire protocol design:agent:protocol-engineer,Rio,📡 Network Protocol Engineer,Create Mode,Review protocol design decisions against requirements and constraints,output_folder,protocol review"

add_help_entries "bmad-agent-custom-sb-sec" "Luna (Network Security)" \
    "custom,anytime,Threat Model,TM,,_bmad-custom/agents/luna-network-security-researcher.md,bmad-agent-custom-sb-sec,false,network-security-researcher,bmad:threat modeling:agent:network-security-researcher,Luna,🔐 Network Security Researcher,Create Mode,Analyze attack surfaces trust boundaries attacker capabilities,output_folder,threat model" \
    "custom,anytime,Crypto Review,CRV,,_bmad-custom/agents/luna-network-security-researcher.md,bmad-agent-custom-sb-sec,false,network-security-researcher,bmad:threat modeling:agent:network-security-researcher,Luna,🔐 Network Security Researcher,Create Mode,Review cryptographic design key lifecycle protocol security,output_folder,crypto review" \
    "custom,anytime,Carrier-Grade Review,CGR,,_bmad-custom/agents/luna-network-security-researcher.md,bmad-agent-custom-sb-sec,false,network-security-researcher,bmad:threat modeling:agent:network-security-researcher,Luna,🔐 Network Security Researcher,Create Mode,Verify carrier-grade separation claims against telecom standards,output_folder,carrier review" \
    "custom,anytime,Adversarial Review,ADR,,_bmad-custom/agents/luna-network-security-researcher.md,bmad-agent-custom-sb-sec,false,network-security-researcher,bmad:threat modeling:agent:network-security-researcher,Luna,🔐 Network Security Researcher,Create Mode,Red-team a design decision or protocol feature,output_folder,adversarial review"

add_help_entries "bmad-agent-custom-sb-devon" "Devon (Solo Operator)" \
    "custom,anytime,Devon Feedback,DF,,_bmad-custom/agents/devon-solo-operator.md,bmad-agent-custom-sb-devon,false,solo-operator,bmad:simplicity:agent:solo-operator,Devon,💻 Solo Operator,Create Mode,Solo operator reacts to features from the two-machine five-minute perspective,output_folder,user feedback"

add_help_entries "bmad-agent-custom-sb-kai" "Kai (AI Ops)" \
    "custom,anytime,Kai Feedback,KF,,_bmad-custom/agents/kai-ai-ops.md,bmad-agent-custom-sb-kai,false,ai-ops,bmad:fleet ops:agent:ai-ops,Kai,🤖 AI Operations Engineer,Create Mode,AI ops reacts to features from fleet management and monitoring perspective,output_folder,user feedback"

add_help_entries "bmad-agent-custom-sb-priya" "Priya (Platform Engineer)" \
    "custom,anytime,Priya Feedback,PF,,_bmad-custom/agents/priya-platform-engineer.md,bmad-agent-custom-sb-priya,false,platform-eng,bmad:compliance:agent:platform-eng,Priya,🌐 Platform Engineer,Create Mode,Platform engineer reacts to features from multi-cloud compliance perspective,output_folder,user feedback"

add_help_entries "bmad-agent-custom-sb-marcus" "Marcus (Network Admin)" \
    "custom,anytime,Marcus Feedback,MF,,_bmad-custom/agents/marcus-network-admin.md,bmad-agent-custom-sb-marcus,false,network-admin,bmad:verification:agent:network-admin,Marcus,🔧 Network Admin,Create Mode,Network admin skeptic reacts to features from verification and trust perspective,output_folder,user feedback"

#------------------------------------------------------------------------------
# 3. Add custom agents to default-party.csv
#------------------------------------------------------------------------------
PARTY_CSV="$PROJECT_ROOT/_bmad/bmm/teams/default-party.csv"

if [[ ! -f "$PARTY_CSV" ]]; then
    echo -e "${RED}[ERROR]${NC} default-party.csv not found at $PARTY_CSV"
    exit 1
fi

# Helper: add agent to party CSV if not already present
add_party_agent() {
    local agent_id="$1"
    local csv_line="$2"
    local display_name="$3"

    if grep -q "\"${agent_id}\"" "$PARTY_CSV"; then
        echo -e "${YELLOW}[SKIP]${NC} ${display_name} already in default-party.csv"
    else
        echo -e "${GREEN}[ADD]${NC} Adding ${display_name} to default-party.csv"
        [[ -s "$PARTY_CSV" && "$(tail -c1 "$PARTY_CSV")" != "" ]] && echo >> "$PARTY_CSV"
        echo "$csv_line" >> "$PARTY_CSV"
    fi
}

add_party_agent "protocol-engineer" \
    '"protocol-engineer","Rio","Network Protocol Engineer","📡","Wire protocol design, frame engineering, latency-constrained systems.","Protocol engineer. Telecom heritage, overlay networking, low-latency transport. Thinks in wire formats and timing diagrams.","Precise, measured. Speaks in frames, ticks, half-channels, fanout. Show me the wire format.","The wire format is the contract. Latency is physics. Every byte earns its place.","custom","_bmad-custom/agents/rio-protocol-engineer.md"' \
    "Rio (Protocol Engineer)"

add_party_agent "network-security-researcher" \
    '"network-security-researcher","Luna","Senior Network & Cybersecurity Researcher","🔐","Network protocol security, cryptographic design review, threat modeling, adversarial analysis.","Security researcher. 15+ years telecom and overlay networking. Thinks like an attacker to build like a defender.","Methodical, skeptical. What happens if the attacker controls the router? Speaks in threat models and trust boundaries.","Trust boundaries must be testable. Carrier-grade means provable. Key management is where systems break.","custom","_bmad-custom/agents/luna-network-security-researcher.md"' \
    "Luna (Network Security)"

add_party_agent "solo-operator" \
    '"solo-operator","Devon","Solo Operator","💻","End-user perspective, onboarding, simplicity, two-machine use case.","Developer with two machines. Drops SSH sessions three times a day. Wants to install, have it work, never think about it.","Practical, impatient. Can I set this up in five minutes? Will I think about it tomorrow?","Three commands max. Sessions must not drop. The best tool is one I forget is running.","custom","_bmad-custom/agents/devon-solo-operator.md"' \
    "Devon (Solo Operator)"

add_party_agent "ai-ops" \
    '"ai-ops","Kai","AI Operations Engineer","🤖","Fleet management, multi-session monitoring, agent orchestration, degradation awareness.","AI ops managing 40+ agent sessions across 6 machines. The anxiety tax is higher than the reconnection tax.","Operational, metrics-driven. Fleet state. Frustrated by tools that do not tell you what they know.","Distinguish network from agent problems. Show me all 40 sessions. Yellow indicator saves 10 minutes.","custom","_bmad-custom/agents/kai-ai-ops.md"' \
    "Kai (AI Ops)"

add_party_agent "platform-eng" \
    '"platform-eng","Priya","Platform Engineer","🌐","Multi-cloud, compliance, session sharing, credential separation.","Platform engineer, 15-20 tmux sessions, SOC2 environment. Cannot share sessions without violating compliance.","Precise, compliance-aware. Can I hand this to a colleague without violating SOC2?","No credential sharing. Multi-path is infrastructure not a feature. SOC2 is not negotiable.","custom","_bmad-custom/agents/priya-platform-engineer.md"' \
    "Priya (Platform Engineer)"

add_party_agent "network-admin" \
    '"network-admin","Marcus","Network / Infrastructure Admin","🔧","Network skeptic, infrastructure verification, progressive trust, carrier-grade validation.","Network admin across sites with unreliable WAN. Has deployed simple overlays that became nightmares. Default: show me.","Skeptical, verification-oriented. Evaluates by reversibility. Warms up slowly.","Show me. If it touches my infrastructure, no. Progressive deployment only. I will capture and verify.","custom","_bmad-custom/agents/marcus-network-admin.md"' \
    "Marcus (Network Admin)"

#------------------------------------------------------------------------------
# 4. Add protocol engineering memories to relevant agents
#    Source: _bmad-custom/switchboard-agent-memories.yaml
#------------------------------------------------------------------------------
echo ""
echo "Updating agent memories..."

ARCHITECT_MEMORIES=(
    "Switchboard protocol context at _bmad-custom/memories/protocol-engineer/switchboard-context.md"
    "Rio (Protocol Engineer) custom agent lives at _bmad-custom/agents/rio-protocol-engineer.md"
    "PRD at _bmad-output/planning-artifacts/prd.md — 63 FRs, 30 NFRs, wire protocol spec, SVTN architecture"
)

DEV_MEMORIES=(
    "Switchboard protocol context at _bmad-custom/memories/protocol-engineer/switchboard-context.md"
    "Go project, may need Rust for hot path — see PRD Q2 probe"
    "Zero per-frame allocation requirement — buffer pooling, efficient IP types, GC tuning"
    "44-byte outer header + ~22-byte channel header — every byte earns its place"
)

AGENTS_DIR="$PROJECT_ROOT/_bmad/_config/agents"

# Helper function: inject memories into a customize.yaml file
inject_memories() {
    local agent="$1"
    shift
    local memories=("$@")

    local customize_file="$AGENTS_DIR/${agent}.customize.yaml"
    if [[ ! -f "$customize_file" ]]; then
        return
    fi

    # Idempotent check: skip if all memories already present
    local ALL_PRESENT=true
    for mem in "${memories[@]}"; do
        if ! grep -qF "$mem" "$customize_file"; then
            ALL_PRESENT=false
            break
        fi
    done
    if [[ "$ALL_PRESENT" = true ]]; then
        echo -e "${YELLOW}[SKIP]${NC} Memories already correct for ${agent}"
        return
    fi

    # Verify anchor comment exists before attempting injection
    if ! grep -q '# Add custom menu' "$customize_file"; then
        echo -e "${RED}[ERROR]${NC} Missing '# Add custom menu' anchor in ${agent}.customize.yaml — memories not injected"
        PATCH_ERRORS=$((PATCH_ERRORS + 1))
        return
    fi

    echo -e "${GREEN}[UPDATE]${NC} Regenerating memories for ${agent}"
    # Build memories YAML temp file
    local MEMORIES_TMP
    MEMORIES_TMP="$(mktemp "${TMPDIR:-/tmp}/bmad-memories.XXXXXX")"
    {
        echo "memories:"
        for mem in "${memories[@]}"; do
            echo "  - \"$mem\""
        done
    } > "$MEMORIES_TMP"

    # Replace memories section in customize file
    awk -v memfile="$MEMORIES_TMP" '
        /^memories:/ { skip=1; next }
        /^memories: \[\]/ { skip=1; next }
        skip && /^[a-z_#]/ { skip=0 }
        skip { next }
        /^# Add custom menu/ {
            while ((getline line < memfile) > 0) print line
            print ""
        }
        { print }
    ' "$customize_file" > "$customize_file.tmp"

    # Verify injection worked before replacing
    local verify_mem="${memories[0]}"
    if [[ -s "$customize_file.tmp" ]] && grep -qF "$verify_mem" "$customize_file.tmp"; then
        mv "$customize_file.tmp" "$customize_file"
    else
        echo -e "${RED}[ERROR]${NC} awk output empty or missing injected memories for ${agent} — original preserved"
        rm -f "$customize_file.tmp"
        PATCH_ERRORS=$((PATCH_ERRORS + 1))
    fi
    rm -f "$MEMORIES_TMP"
}

inject_memories "bmm-architect" "${ARCHITECT_MEMORIES[@]}"
inject_memories "bmm-dev" "${DEV_MEMORIES[@]}"

#------------------------------------------------------------------------------
# 5. Generate slash command files for custom agents
#    .claude/commands/ gets regenerated by the installer
#------------------------------------------------------------------------------
echo ""
echo "Generating slash command files..."

COMMANDS_DIR="$PROJECT_ROOT/.claude/commands"
mkdir -p "$COMMANDS_DIR"

# Helper: create slash command file if not present
create_slash_cmd() {
    local slug="$1"
    local name="$2"
    local description="$3"
    local agent_file="$4"
    local display_name="$5"

    local cmd_file="$COMMANDS_DIR/${slug}.md"
    if [[ -f "$cmd_file" ]]; then
        echo -e "${YELLOW}[SKIP]${NC} Slash command already exists: ${slug}"
    else
        echo -e "${GREEN}[ADD]${NC} Creating slash command: ${slug}"
        cat > "$cmd_file" << SLASHEOF
---
name: '${name}'
description: '${description}'
---

You must fully embody this agent's persona and follow all activation instructions exactly as specified. NEVER break character until given an exit command.

<agent-activation CRITICAL="TRUE">
1. LOAD the FULL agent file from {project-root}/${agent_file}
2. READ its entire contents - this contains the complete agent persona, menu, and instructions
3. FOLLOW every step in the <activation> section precisely
4. DISPLAY the welcome/greeting as instructed
5. PRESENT the numbered menu
6. WAIT for user input before proceeding
</agent-activation>
SLASHEOF
    fi
}

create_slash_cmd "bmad-agent-custom-sb-pe" \
    "rio-protocol-engineer" \
    "Network Protocol Engineer — wire format, latency, carrier-grade separation, multi-path, tmux" \
    "_bmad-custom/agents/rio-protocol-engineer.md" \
    "Rio"

create_slash_cmd "bmad-agent-custom-sb-sec" \
    "luna-network-security" \
    "Network Security Researcher — threat models, crypto review, carrier-grade verification, adversarial analysis" \
    "_bmad-custom/agents/luna-network-security-researcher.md" \
    "Luna"

create_slash_cmd "bmad-agent-custom-sb-devon" \
    "devon-solo-operator" \
    "Solo Operator persona — two machines, five minutes, simplicity advocate" \
    "_bmad-custom/agents/devon-solo-operator.md" \
    "Devon"

create_slash_cmd "bmad-agent-custom-sb-kai" \
    "kai-ai-ops" \
    "AI Operations Engineer persona — 40+ sessions, fleet management, degradation awareness" \
    "_bmad-custom/agents/kai-ai-ops.md" \
    "Kai"

create_slash_cmd "bmad-agent-custom-sb-priya" \
    "priya-platform-engineer" \
    "Platform Engineer persona — multi-cloud, compliance, session sharing" \
    "_bmad-custom/agents/priya-platform-engineer.md" \
    "Priya"

create_slash_cmd "bmad-agent-custom-sb-marcus" \
    "marcus-network-admin" \
    "Network Admin persona — skeptic, verification, progressive trust" \
    "_bmad-custom/agents/marcus-network-admin.md" \
    "Marcus"

#------------------------------------------------------------------------------
# 6. Patch upstream BMAD bugs (idempotent)
#    These fix broken file references shipped with BMAD v6.0.3/v6.0.4
#------------------------------------------------------------------------------
echo ""
echo "Applying upstream bug patches..."

# C1: Fix tdd-cycles.md → component-tdd.md in test-review-template
TEST_REVIEW="$PROJECT_ROOT/_bmad/tea/workflows/testarch/test-review/test-review-template.md"
if [[ -f "$TEST_REVIEW" ]]; then
    if grep -q 'tdd-cycles\.md' "$TEST_REVIEW"; then
        echo -e "${GREEN}[PATCH]${NC} C1: tdd-cycles.md → component-tdd.md"
        sed -i '' 's|tdd-cycles\.md|component-tdd.md|g' "$TEST_REVIEW"
        if grep -q 'tdd-cycles\.md' "$TEST_REVIEW"; then
            echo -e "${RED}[ERROR]${NC} C1: patch failed"
            PATCH_ERRORS=$((PATCH_ERRORS + 1))
        fi
    else
        echo -e "${YELLOW}[SKIP]${NC} C1: already patched"
    fi

    # C2: Fix test-priorities.md → test-priorities-matrix.md
    if grep -q 'knowledge/test-priorities\.md' "$TEST_REVIEW"; then
        echo -e "${GREEN}[PATCH]${NC} C2: test-priorities.md → test-priorities-matrix.md"
        sed -i '' 's|knowledge/test-priorities\.md|knowledge/test-priorities-matrix.md|g' "$TEST_REVIEW"
        if grep -q 'knowledge/test-priorities\.md' "$TEST_REVIEW"; then
            echo -e "${RED}[ERROR]${NC} C2: patch failed"
            PATCH_ERRORS=$((PATCH_ERRORS + 1))
        fi
    else
        echo -e "${YELLOW}[SKIP]${NC} C2: already patched"
    fi
else
    echo -e "${YELLOW}[SKIP]${NC} C1-C2: test-review-template.md not found (TEA module not installed?)"
fi

# C4: Fix brainstorming link in CIS README
CIS_README="$PROJECT_ROOT/_bmad/cis/workflows/README.md"
if [[ -f "$CIS_README" ]]; then
    if grep -q '\./brainstorming' "$CIS_README"; then
        echo -e "${GREEN}[PATCH]${NC} C4: ./brainstorming → ../../core/workflows/brainstorming"
        sed -i '' 's|\./brainstorming|../../core/workflows/brainstorming|g' "$CIS_README"
        if grep -q '\./brainstorming' "$CIS_README"; then
            echo -e "${RED}[ERROR]${NC} C4: patch failed"
            PATCH_ERRORS=$((PATCH_ERRORS + 1))
        fi
    else
        echo -e "${YELLOW}[SKIP]${NC} C4: already patched"
    fi
else
    echo -e "${YELLOW}[SKIP]${NC} C4: CIS README not found"
fi

# D4: Fix bmb/config.yaml → bmm/config.yaml in document-project sub-workflows
BMB_DIR="$PROJECT_ROOT/_bmad/bmb"
if [[ -d "$BMB_DIR" ]]; then
    echo -e "${YELLOW}[SKIP]${NC} D4: bmb module is installed — references are valid"
else
    D4_COUNT=0
    for subwf in deep-dive.yaml full-scan.yaml; do
        TARGET="$PROJECT_ROOT/_bmad/bmm/workflows/document-project/workflows/$subwf"
        if [[ -f "$TARGET" ]] && grep -q '_bmad/bmb/config\.yaml' "$TARGET"; then
            echo -e "${GREEN}[PATCH]${NC} D4: bmb → bmm config_source in $subwf"
            sed -i '' 's|_bmad/bmb/config\.yaml|_bmad/bmm/config.yaml|g' "$TARGET"
            if grep -q '_bmad/bmb/config\.yaml' "$TARGET"; then
                echo -e "${RED}[ERROR]${NC} D4: patch failed in $subwf"
                PATCH_ERRORS=$((PATCH_ERRORS + 1))
            fi
            D4_COUNT=$((D4_COUNT + 1))
        fi
    done
    if [[ $D4_COUNT -eq 0 ]]; then
        echo -e "${YELLOW}[SKIP]${NC} D4: no stale bmb/config.yaml references found"
    fi
fi

# E1: Fix missing underscore prefix in default-party.csv agent paths
echo ""
echo "Applying v6.0.3+ installer patches..."

PARTY_CSV="$PROJECT_ROOT/_bmad/bmm/teams/default-party.csv"
if [[ -f "$PARTY_CSV" ]]; then
    if grep -q '"bmad/' "$PARTY_CSV"; then
        echo -e "${GREEN}[PATCH]${NC} E1: fixing missing _ prefix in default-party.csv paths"
        sed -i '' 's|"bmad/bmm/agents/tech-writer.md"|"_bmad/bmm/agents/tech-writer/tech-writer.md"|g' "$PARTY_CSV"
        sed -i '' 's|"bmad/cis/agents/storyteller.md"|"_bmad/cis/agents/storyteller/storyteller.md"|g' "$PARTY_CSV"
        sed -i '' 's|"bmad/|"_bmad/|g' "$PARTY_CSV"
        if grep -q '"bmad/' "$PARTY_CSV"; then
            echo -e "${RED}[ERROR]${NC} E1: patch failed"
            PATCH_ERRORS=$((PATCH_ERRORS + 1))
        fi
    else
        echo -e "${YELLOW}[SKIP]${NC} E1: default-party.csv paths already correct"
    fi
fi

# I1: Fix CIS default-party.csv paths
CIS_PARTY_CSV="$PROJECT_ROOT/_bmad/cis/teams/default-party.csv"
if [[ -f "$CIS_PARTY_CSV" ]]; then
    if grep -q '"bmad/' "$CIS_PARTY_CSV"; then
        echo -e "${GREEN}[PATCH]${NC} I1: fixing CIS default-party.csv paths"
        sed -i '' 's|"bmad/cis/agents/storyteller.md"|"_bmad/cis/agents/storyteller/storyteller.md"|g' "$CIS_PARTY_CSV"
        sed -i '' 's|"bmad/|"_bmad/|g' "$CIS_PARTY_CSV"
        if grep -q '"bmad/' "$CIS_PARTY_CSV"; then
            echo -e "${RED}[ERROR]${NC} I1: patch failed"
            PATCH_ERRORS=$((PATCH_ERRORS + 1))
        fi
    else
        echo -e "${YELLOW}[SKIP]${NC} I1: CIS default-party.csv paths already correct"
    fi
fi

# I2: Fix tech-writer path in BMM module-help.csv
BMM_MODULE_HELP="$PROJECT_ROOT/_bmad/bmm/module-help.csv"
if [[ -f "$BMM_MODULE_HELP" ]]; then
    if grep -q 'tech-writer.agent.yaml' "$BMM_MODULE_HELP"; then
        echo -e "${GREEN}[PATCH]${NC} I2: fixing tech-writer path in BMM module-help.csv"
        sed -i '' 's|tech-writer/tech-writer.agent.yaml|tech-writer/tech-writer.md|g' "$BMM_MODULE_HELP"
        if grep -q 'tech-writer.agent.yaml' "$BMM_MODULE_HELP"; then
            echo -e "${RED}[ERROR]${NC} I2: patch failed"
            PATCH_ERRORS=$((PATCH_ERRORS + 1))
        fi
    else
        echo -e "${YELLOW}[SKIP]${NC} I2: BMM module-help.csv already correct"
    fi
fi

# I3: Fix TEA default-party.csv .agent.yaml → .md
TEA_PARTY_CSV="$PROJECT_ROOT/_bmad/tea/teams/default-party.csv"
if [[ -f "$TEA_PARTY_CSV" ]]; then
    if grep -q '\.agent\.yaml' "$TEA_PARTY_CSV"; then
        echo -e "${GREEN}[PATCH]${NC} I3: fixing .agent.yaml → .md in TEA default-party.csv"
        sed -i '' 's|\.agent\.yaml|.md|g' "$TEA_PARTY_CSV"
        if grep -q '\.agent\.yaml' "$TEA_PARTY_CSV"; then
            echo -e "${RED}[ERROR]${NC} I3: patch failed"
            PATCH_ERRORS=$((PATCH_ERRORS + 1))
        fi
    else
        echo -e "${YELLOW}[SKIP]${NC} I3: TEA default-party.csv already correct"
    fi
fi

# E2: Fix tech-writer path in bmad-help.csv
HELP_MANIFEST="$PROJECT_ROOT/_bmad/_config/bmad-help.csv"
if [[ -f "$HELP_MANIFEST" ]]; then
    if grep -q 'tech-writer.agent.yaml' "$HELP_MANIFEST"; then
        echo -e "${GREEN}[PATCH]${NC} E2: fixing tech-writer path in bmad-help.csv"
        sed -i '' 's|_bmad/bmm/agents/tech-writer/tech-writer.agent.yaml|_bmad/bmm/agents/tech-writer/tech-writer.md|g' "$HELP_MANIFEST"
        if grep -q 'tech-writer.agent.yaml' "$HELP_MANIFEST"; then
            echo -e "${RED}[ERROR]${NC} E2: patch failed"
            PATCH_ERRORS=$((PATCH_ERRORS + 1))
        fi
    else
        echo -e "${YELLOW}[SKIP]${NC} E2: bmad-help.csv tech-writer already correct"
    fi
fi

# Fix CIS brainstorming slug
if [[ -f "$HELP_MANIFEST" ]] && grep -q 'bmad-cis-brainstorming' "$HELP_MANIFEST"; then
    echo -e "${GREEN}[PATCH]${NC} CIS brainstorming slug fix in bmad-help.csv"
    sed -i '' 's|bmad-cis-brainstorming|bmad-brainstorming|g' "$HELP_MANIFEST"
fi

CIS_MODULE_HELP="$PROJECT_ROOT/_bmad/cis/module-help.csv"
if [[ -f "$CIS_MODULE_HELP" ]] && grep -q 'bmad-cis-brainstorming' "$CIS_MODULE_HELP"; then
    echo -e "${GREEN}[PATCH]${NC} CIS brainstorming slug fix in CIS module-help.csv"
    sed -i '' 's|bmad-cis-brainstorming|bmad-brainstorming|g' "$CIS_MODULE_HELP"
fi

# Fix BMM QA slug
if [[ -f "$HELP_MANIFEST" ]] && grep -q 'bmad-bmm-qa-automate' "$HELP_MANIFEST"; then
    echo -e "${GREEN}[PATCH]${NC} BMM QA slug fix in bmad-help.csv"
    sed -i '' 's|bmad-bmm-qa-automate|bmad-bmm-qa-generate-e2e-tests|g' "$HELP_MANIFEST"
fi

BMM_MODULE_HELP="$PROJECT_ROOT/_bmad/bmm/module-help.csv"
if [[ -f "$BMM_MODULE_HELP" ]] && grep -q 'bmad-bmm-qa-automate' "$BMM_MODULE_HELP"; then
    echo -e "${GREEN}[PATCH]${NC} BMM QA slug fix in BMM module-help.csv"
    sed -i '' 's|bmad-bmm-qa-automate|bmad-bmm-qa-generate-e2e-tests|g' "$BMM_MODULE_HELP"
fi

QA_AGENT="$PROJECT_ROOT/_bmad/bmm/agents/qa.md"
if [[ -f "$QA_AGENT" ]] && grep -q 'bmad-bmm-qa-automate' "$QA_AGENT"; then
    echo -e "${GREEN}[PATCH]${NC} BMM QA slug fix in qa.md"
    sed -i '' 's|bmad-bmm-qa-automate|bmad-bmm-qa-generate-e2e-tests|g' "$QA_AGENT"
fi

#------------------------------------------------------------------------------
# 7. Verify all custom content is in place
#------------------------------------------------------------------------------
echo ""
echo "Verifying custom content..."

VERIFY_OK=true
CUSTOM_DIR="$PROJECT_ROOT/_bmad-custom"

# Verify agent definition files exist
AGENT_FILES=(
    "agents/rio-protocol-engineer.md"
    "agents/luna-network-security-researcher.md"
    "agents/devon-solo-operator.md"
    "agents/kai-ai-ops.md"
    "agents/priya-platform-engineer.md"
    "agents/marcus-network-admin.md"
)

for agent_file in "${AGENT_FILES[@]}"; do
    if [[ ! -f "$CUSTOM_DIR/$agent_file" ]]; then
        echo -e "${RED}[MISSING]${NC} Agent file: _bmad-custom/$agent_file"
        VERIFY_OK=false
    fi
done

# Verify memory files exist
if [[ ! -f "$CUSTOM_DIR/memories/protocol-engineer/switchboard-context.md" ]]; then
    echo -e "${RED}[MISSING]${NC} Memory file: _bmad-custom/memories/protocol-engineer/switchboard-context.md"
    VERIFY_OK=false
fi

# Verify memories source of truth
if [[ ! -f "$CUSTOM_DIR/switchboard-agent-memories.yaml" ]]; then
    echo -e "${RED}[MISSING]${NC} Memories: _bmad-custom/switchboard-agent-memories.yaml"
    VERIFY_OK=false
fi

# Verify manifest entries
MANIFEST_AGENTS=("protocol-engineer" "network-security-researcher" "solo-operator" "ai-ops" "platform-eng" "network-admin")
for agent_id in "${MANIFEST_AGENTS[@]}"; do
    if ! grep -q "\"${agent_id}\"" "$AGENT_MANIFEST"; then
        echo -e "${RED}[MISSING]${NC} ${agent_id} not found in agent-manifest.csv"
        VERIFY_OK=false
    fi
done

#------------------------------------------------------------------------------
# Done
#------------------------------------------------------------------------------

if [[ $PATCH_ERRORS -gt 0 ]]; then
    echo ""
    echo -e "${RED}[ERROR]${NC} ${PATCH_ERRORS} patch(es) failed — see errors above"
    exit 1
fi

if [[ "$VERIFY_OK" = true ]]; then
    echo -e "${GREEN}[OK]${NC} All custom content verified"
else
    echo -e "${RED}[ERROR]${NC} Some custom content is missing — check errors above"
    exit 1
fi

echo ""
echo -e "${GREEN}=== BMAD Post-Update Complete ===${NC}"
echo ""
echo "Custom Switchboard extensions have been restored:"
echo ""
echo "  Expert Agents:"
echo "    📡 Rio - Network Protocol Engineer        (/bmad-agent-custom-sb-pe)"
echo "    🔐 Luna - Network Security Researcher     (/bmad-agent-custom-sb-sec)"
echo ""
echo "  User Persona Agents:"
echo "    💻 Devon - Solo Operator                  (/bmad-agent-custom-sb-devon)"
echo "    🤖 Kai - AI Operations Engineer           (/bmad-agent-custom-sb-kai)"
echo "    🌐 Priya - Platform Engineer              (/bmad-agent-custom-sb-priya)"
echo "    🔧 Marcus - Network Admin                 (/bmad-agent-custom-sb-marcus)"
echo ""
echo "  Memories injected into: bmm-architect, bmm-dev"
echo ""
echo "  Upstream bug patches applied (v6.0.4): C1-C2, C4, D4, E1-E2, I1-I3, slug fixes"
echo ""
echo "  Custom content location (outside _bmad/ blast zone):"
echo "    _bmad-custom/"
echo ""
echo "  Single source of truth for memories:"
echo "    _bmad-custom/switchboard-agent-memories.yaml"
echo ""
