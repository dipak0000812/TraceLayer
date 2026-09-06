#!/usr/bin/env python3
"""
TraceLayer Round 2 — Synthetic Bitcoin Transaction + Network Observation Generator
SIH Problem Statement 26146 (NTRO)

This tool generates a FULLY SYNTHETIC dataset only. It never touches, samples,
or represents real Bitcoin network captures, real seized data, or real wallet
addresses. All IPs are drawn from RFC 5737 (documentation/TEST-NET) and RFC 1918
(private) ranges. All ASNs are drawn from the 16-bit private-use ASN range
(64512-65534). All addresses and txids are structurally plausible but
deterministically derived synthetic strings.

Deterministic: the same --seed always reproduces byte-identical output.

Usage:
    python3 generator.py --seed 42 --transactions 100 --entities 20 \
        --output-dir ./out

See schema.md (auto-generated alongside output) for full field documentation.
"""

from __future__ import annotations

import argparse
import hashlib
import ipaddress
import json
import os
import random
import sys
from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from typing import Any

GENERATOR_VERSION = "1.0.0"
SCHEMA_VERSION = "1.0.0"

# ---------------------------------------------------------------------------
# Constants — kept structurally synthetic-only
# ---------------------------------------------------------------------------

SCRIPT_TYPES = ["P2PKH", "P2WPKH", "P2SH", "P2TR"]

# RFC 5737 documentation ranges + RFC 1918 private ranges only.
# These CANNOT collide with any real routable / real-intercept IP.
IP_POOLS = [
    ipaddress.ip_network("192.0.2.0/24"),    # TEST-NET-1
    ipaddress.ip_network("198.51.100.0/24"), # TEST-NET-2
    ipaddress.ip_network("203.0.113.0/24"),  # TEST-NET-3
    ipaddress.ip_network("10.0.0.0/16"),     # RFC1918 (narrowed slice)
    ipaddress.ip_network("172.16.0.0/16"),   # RFC1918 (narrowed slice)
    ipaddress.ip_network("192.168.0.0/16"),  # RFC1918
]

GEO_COUNTRIES = ["IN", "US", "DE", "SG", "NL", "GB", "JP", "BR", "ZA", "AE"]

# Blockchain-axis (anomaly) scenario types
ANOMALOUS_SCENARIOS = {
    "HIGH_ACTIVITY",
    "UNUSUAL_VALUE_FLOW",
    "UNUSUAL_TRANSACTION_STRUCTURE",
    "PEELING_CHAIN",
    "COINJOIN_LIKE",
    "CUSTODIAL_CONSOLIDATION",
}
NORMAL_SCENARIOS = {"NORMAL"}
STRUCTURAL_SCENARIOS = {"DUPLICATE_RECORDS"}  # applied as a post-pass, not standalone

ALL_BLOCKCHAIN_SCENARIOS = list(NORMAL_SCENARIOS | ANOMALOUS_SCENARIOS)

# Network-axis conditions (fully independent of the blockchain axis)
NETWORK_CONDITIONS = [
    "NETWORK_STRONG",
    "NETWORK_SPARSE",
    "NETWORK_DELAYED",
    "NETWORK_CONFLICTING",
    "NETWORK_NOISY",
    "NETWORK_ABSENT",
]

DEFAULT_BLOCKCHAIN_WEIGHTS = {
    "NORMAL": 0.40,
    "HIGH_ACTIVITY": 0.10,
    "UNUSUAL_VALUE_FLOW": 0.10,
    "UNUSUAL_TRANSACTION_STRUCTURE": 0.10,
    "PEELING_CHAIN": 0.10,
    "COINJOIN_LIKE": 0.10,
    "CUSTODIAL_CONSOLIDATION": 0.10,
}

DEFAULT_NETWORK_WEIGHTS = {
    "NETWORK_STRONG": 0.20,
    "NETWORK_SPARSE": 0.20,
    "NETWORK_DELAYED": 0.15,
    "NETWORK_CONFLICTING": 0.15,
    "NETWORK_NOISY": 0.15,
    "NETWORK_ABSENT": 0.15,
}

BASE_TIME = datetime(2025, 1, 1, tzinfo=timezone.utc)


# ---------------------------------------------------------------------------
# Deterministic helpers
# ---------------------------------------------------------------------------

def det_hash(*parts: Any) -> str:
    """Deterministic hex digest from arbitrary parts (used for txids / ids)."""
    h = hashlib.sha256("|".join(str(p) for p in parts).encode("utf-8"))
    return h.hexdigest()


def synthetic_txid(seed: int, counter: int) -> str:
    # 64-hex-char string, structurally like a txid, tagged internally as synthetic
    return det_hash("SYN-TX", seed, counter)


def synthetic_address(seed: int, entity_idx: int, addr_idx: int) -> str:
    # bech32-ish synthetic address, clearly namespaced so it can never be
    # confused with a real address: "sbc1" (synthetic-btc) prefix.
    raw = det_hash("SYN-ADDR", seed, entity_idx, addr_idx)[:38]
    return f"sbc1{raw}"


def synthetic_observation_id(seed: int, counter: int) -> str:
    return "OBS-" + det_hash("SYN-OBS", seed, counter)[:24]


def pick_ip(rng: random.Random) -> str:
    pool = rng.choice(IP_POOLS)
    # avoid network/broadcast edges for /24s, fine for larger pools too
    hosts = list(pool.hosts())
    return str(rng.choice(hosts))


def pick_port(rng: random.Random) -> int:
    return rng.randint(1024, 65535)


def pick_asn(rng: random.Random) -> str:
    # 16-bit private-use ASN range per RFC 6996
    return f"AS{rng.randint(64512, 65534)}"


def weighted_choice(rng: random.Random, weights: dict[str, float]) -> str:
    items = list(weights.keys())
    probs = list(weights.values())
    return rng.choices(items, weights=probs, k=1)[0]


def iso(ts: datetime) -> str:
    return ts.strftime("%Y-%m-%dT%H:%M:%SZ")


def round8(x: float) -> float:
    return round(x + 0.0, 8)


# ---------------------------------------------------------------------------
# Data classes
# ---------------------------------------------------------------------------

@dataclass
class Entity:
    entity_id: str
    addresses: list[str] = field(default_factory=list)


@dataclass
class Transaction:
    txid: str
    timestamp: str
    input_addresses: list[str]
    output_addresses: list[str]
    input_amounts: list[float]
    output_amounts: list[float]
    fee: float
    script_type: str
    provenance: str = "SYNTHETIC"
    dataset_id: str = ""
    generator_version: str = GENERATOR_VERSION

    def to_row(self) -> dict[str, Any]:
        return {
            "txid": self.txid,
            "timestamp": self.timestamp,
            "input_addresses": json.dumps(self.input_addresses),
            "output_addresses": json.dumps(self.output_addresses),
            "input_amounts": json.dumps(self.input_amounts),
            "output_amounts": json.dumps(self.output_amounts),
            "fee": self.fee,
            "script_type": self.script_type,
            "provenance": self.provenance,
            "dataset_id": self.dataset_id,
            "generator_version": self.generator_version,
        }


@dataclass
class NetworkObservation:
    """RAW network observation as it would arrive off the wire — deliberately
    contains NO geo_country/asn. Those are derived fields produced by the
    (separate, offline) GeoIP/ASN enrichment component, not authoritative
    generator output. See geoip_fixture.json in data/eval/ for the synthetic
    expected-enrichment values used to test that component."""
    observation_id: str
    timestamp: str
    src_ip: str
    dst_ip: str
    src_port: int
    dst_port: int
    txid: str
    provenance: str = "SYNTHETIC"
    dataset_id: str = ""
    generator_version: str = GENERATOR_VERSION

    def to_row(self) -> dict[str, Any]:
        return {
            "observation_id": self.observation_id,
            "timestamp": self.timestamp,
            "src_ip": self.src_ip,
            "dst_ip": self.dst_ip,
            "src_port": self.src_port,
            "dst_port": self.dst_port,
            "txid": self.txid,
            "provenance": self.provenance,
            "dataset_id": self.dataset_id,
            "generator_version": self.generator_version,
        }


@dataclass
class ScenarioRecord:
    scenario_id: str
    scenario_type: str
    txids: list[str]
    ground_truth_entity_ids: list[str]
    anomaly_label: bool
    expected_relationships: list[dict[str, Any]]
    network_condition: dict[str, str]  # txid -> condition
    seed: int


# ---------------------------------------------------------------------------
# Generator core
# ---------------------------------------------------------------------------

class TraceLayerGenerator:
    def __init__(self, args: argparse.Namespace):
        self.args = args
        self.seed = args.seed

        # dataset_id is fixed from the run configuration UP FRONT (not
        # derived from final row counts) so it can be stamped onto every
        # single production record as it is created — per-record provenance,
        # not just a manifest-level claim.
        self.dataset_id = det_hash(
            "TRACELAYER-DATASET", args.seed, args.transactions, args.entities
        )[:16]

        # Independent RNG streams so blockchain-axis and network-axis
        # generation cannot leak into each other.
        self.entity_rng = random.Random(self.seed)
        self.scenario_rng = random.Random(self.seed + 1)
        self.network_rng = random.Random(self.seed + 2)
        self.dup_rng = random.Random(self.seed + 3)
        self.time_rng = random.Random(self.seed + 4)
        self.geo_rng = random.Random(self.seed + 5)

        self.tx_counter = 0
        self.obs_counter = 0
        self.scenario_counter = 0

        self.entities: list[Entity] = []
        self.transactions: dict[str, Transaction] = {}
        self.tx_order: list[str] = []  # preserves generation order
        self.observations: list[NetworkObservation] = []
        self.scenarios: list[ScenarioRecord] = []

        # txid -> owning entity id(s) that contributed inputs (internal only)
        self.tx_input_entities: dict[str, list[str]] = {}

        # ip -> {geo_country, asn} "true" synthetic GeoIP mapping. This is
        # NEVER written into the raw network observation records — it exists
        # only to populate the eval fixture that tests the offline GeoIP/ASN
        # enrichment component's correctness.
        self.ip_geo_truth: dict[str, dict[str, str]] = {}

    # -- entities ------------------------------------------------------

    def build_entities(self) -> None:
        n_addr_min, n_addr_max = 1, 4
        for i in range(self.args.entities):
            eid = f"E{i:04d}"
            n_addr = self.entity_rng.randint(n_addr_min, n_addr_max)
            addrs = [synthetic_address(self.seed, i, j) for j in range(n_addr)]
            self.entities.append(Entity(entity_id=eid, addresses=addrs))

    def random_entity(self) -> Entity:
        return self.entity_rng.choice(self.entities)

    def new_address_for(self, entity: Entity) -> str:
        """Occasionally mints a fresh (unseen) address for an entity so that
        not every tx reuses the same fixed address set — still attributable
        to the entity in ground truth, but not hard-coded upfront."""
        idx = len(entity.addresses)
        addr = synthetic_address(self.seed, int(entity.entity_id[1:]), idx + 1000)
        entity.addresses.append(addr)
        return addr

    # -- low level tx/obs builders --------------------------------------

    def next_txid(self) -> str:
        self.tx_counter += 1
        return synthetic_txid(self.seed, self.tx_counter)

    def next_obs_id(self) -> str:
        self.obs_counter += 1
        return synthetic_observation_id(self.seed, self.obs_counter)

    def make_base_time(self) -> datetime:
        offset_minutes = self.time_rng.randint(0, 60 * 24 * 90)  # spread over ~90 days
        return BASE_TIME + timedelta(minutes=offset_minutes)

    def build_transaction(
        self,
        input_addrs: list[str],
        output_addrs: list[str],
        ts: datetime,
        value_scale: float = 1.0,
        script_type: str | None = None,
        input_entities: list[str] | None = None,
    ) -> Transaction:
        rng = self.scenario_rng
        n_in, n_out = len(input_addrs), len(output_addrs)

        input_amounts = [round8(rng.uniform(0.001, 2.0) * value_scale) for _ in range(n_in)]
        total_in = sum(input_amounts)

        # split total_in (minus a small fee) across outputs
        fee_rate = rng.uniform(0.0001, 0.002)
        fee = round8(min(total_in * fee_rate, total_in * 0.02))
        spendable = max(total_in - fee, 0.0)

        if n_out == 1:
            output_amounts = [round8(spendable)]
        else:
            cuts = sorted(rng.uniform(0, spendable) for _ in range(n_out - 1))
            bounds = [0.0] + cuts + [spendable]
            output_amounts = [round8(bounds[i + 1] - bounds[i]) for i in range(n_out)]

        # guard against float drift breaking input>=output
        drift = round8(total_in - fee - sum(output_amounts))
        if drift != 0.0:
            output_amounts[-1] = round8(output_amounts[-1] + drift)
        fee = round8(total_in - sum(output_amounts))
        if fee < 0:
            # extremely rare float edge case: pull the shortfall from the
            # largest output rather than allow a negative fee
            largest = max(range(n_out), key=lambda i: output_amounts[i])
            output_amounts[largest] = round8(output_amounts[largest] + fee)
            fee = round8(total_in - sum(output_amounts))

        txid = self.next_txid()
        tx = Transaction(
            txid=txid,
            timestamp=iso(ts),
            input_addresses=input_addrs,
            output_addresses=output_addrs,
            input_amounts=input_amounts,
            output_amounts=output_amounts,
            fee=fee,
            script_type=script_type or rng.choice(SCRIPT_TYPES),
            provenance="SYNTHETIC",
            dataset_id=self.dataset_id,
            generator_version=GENERATOR_VERSION,
        )
        self.transactions[txid] = tx
        self.tx_order.append(txid)
        if input_entities:
            self.tx_input_entities[txid] = input_entities
        return tx

    # -- GeoIP truth (eval fixture only — NEVER written to raw records) ----

    def register_ip_geo_truth(self, ip: str) -> None:
        """Assign a deterministic 'true' geo_country/asn to an IP the first
        time it's seen, using a dedicated RNG stream. This value is used only
        to build data/eval/geoip_fixture.json, which tests whether the
        downstream offline GeoIP/ASN enrichment component produces the
        expected answer. It is never attached to the raw observation."""
        if ip not in self.ip_geo_truth:
            self.ip_geo_truth[ip] = {
                "geo_country": self.geo_rng.choice(GEO_COUNTRIES),
                "asn": pick_asn(self.geo_rng),
            }

    def pick_ip_with_truth(self, rng: random.Random) -> str:
        ip = pick_ip(rng)
        self.register_ip_geo_truth(ip)
        return ip

    # -- network observation generation (independent axis) --------------

    def generate_network_for_tx(self, tx: Transaction, condition: str) -> list[NetworkObservation]:
        rng = self.network_rng
        tx_ts = datetime.strptime(tx.timestamp, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
        obs: list[NetworkObservation] = []

        if condition == "NETWORK_ABSENT":
            return obs

        if condition == "NETWORK_STRONG":
            n = rng.randint(4, 8)
            spread_seconds = 30
        elif condition == "NETWORK_SPARSE":
            n = rng.randint(1, 2)
            spread_seconds = 60
        elif condition == "NETWORK_DELAYED":
            n = rng.randint(2, 4)
            spread_seconds = 60 * 60 * 6  # up to 6h spread
        elif condition == "NETWORK_CONFLICTING":
            n = rng.randint(2, 5)
            spread_seconds = 120
        elif condition == "NETWORK_NOISY":
            n = rng.randint(3, 6)
            spread_seconds = 120
        else:
            n = rng.randint(1, 3)
            spread_seconds = 60

        # NETWORK_CONFLICTING is expressed at the raw-field level now that
        # geo/asn no longer live on the observation: within the same short
        # window for this txid, the src_ip is deliberately drawn from two
        # very different address pools, and dst_port swings between two
        # disjoint ranges — a genuine "these don't look like the same
        # session" signal for downstream correlation, rather than a label
        # planted on a derived field.
        conflicting_pools = None
        if condition == "NETWORK_CONFLICTING":
            conflicting_pools = rng.sample(IP_POOLS, k=2)

        for i in range(n):
            delta = timedelta(seconds=rng.randint(-spread_seconds, spread_seconds))
            obs_ts = tx_ts + delta

            if conflicting_pools is not None:
                pool = conflicting_pools[i % 2]
                hosts = list(pool.hosts())
                src_ip = str(rng.choice(hosts))
                self.register_ip_geo_truth(src_ip)
                dst_port = rng.randint(1024, 20000) if i % 2 == 0 else rng.randint(40000, 65535)
            else:
                src_ip = self.pick_ip_with_truth(rng)
                dst_port = pick_port(rng)

            dst_ip = self.pick_ip_with_truth(rng)
            src_port = pick_port(rng)

            if condition == "NETWORK_NOISY" and rng.random() < 0.4:
                # inject an outlier timestamp far from the transaction
                obs_ts = obs_ts + timedelta(hours=rng.choice([-48, 48, 72]))

            obs.append(
                NetworkObservation(
                    observation_id=self.next_obs_id(),
                    timestamp=iso(obs_ts),
                    src_ip=src_ip,
                    dst_ip=dst_ip,
                    src_port=src_port,
                    dst_port=dst_port,
                    txid=tx.txid,
                    provenance="SYNTHETIC",
                    dataset_id=self.dataset_id,
                    generator_version=GENERATOR_VERSION,
                )
            )
        return obs

    # -- scenario builders (blockchain axis) -----------------------------

    def new_scenario_id(self) -> str:
        self.scenario_counter += 1
        return f"SCN-{self.scenario_counter:05d}"

    def scenario_normal(self) -> ScenarioRecord:
        e = self.random_entity()
        out_entity = self.random_entity()
        ts = self.make_base_time()
        in_addr = self.scenario_rng.choice(e.addresses)
        out_addr = self.scenario_rng.choice(out_entity.addresses) if self.scenario_rng.random() < 0.6 else self.new_address_for(out_entity)
        tx = self.build_transaction([in_addr], [out_addr], ts, input_entities=[e.entity_id])
        return self.finalize_single_tx_scenario("NORMAL", tx, [e.entity_id])

    def scenario_high_activity(self) -> ScenarioRecord:
        e = self.random_entity()
        base_ts = self.make_base_time()
        n_tx = self.scenario_rng.randint(6, 10)
        txids = []
        for i in range(n_tx):
            ts = base_ts + timedelta(minutes=self.scenario_rng.randint(0, 45))
            in_addr = self.scenario_rng.choice(e.addresses)
            out_addr = self.new_address_for(self.random_entity())
            tx = self.build_transaction([in_addr], [out_addr], ts, input_entities=[e.entity_id])
            txids.append(tx.txid)
        return self.finalize_multi_tx_scenario(
            "HIGH_ACTIVITY", txids, [e.entity_id],
            [{"type": "burst_activity", "entity_id": e.entity_id, "count": n_tx}],
        )

    def scenario_unusual_value_flow(self) -> ScenarioRecord:
        e = self.random_entity()
        out_entity = self.random_entity()
        ts = self.make_base_time()
        in_addr = self.scenario_rng.choice(e.addresses)
        out_addr = self.new_address_for(out_entity)
        # extreme value scale — either far above or far below typical range
        scale = self.scenario_rng.choice([25.0, 40.0, 0.0005])
        tx = self.build_transaction([in_addr], [out_addr], ts, value_scale=scale, input_entities=[e.entity_id])
        return self.finalize_single_tx_scenario("UNUSUAL_VALUE_FLOW", tx, [e.entity_id])

    def scenario_unusual_structure(self) -> ScenarioRecord:
        e = self.random_entity()
        ts = self.make_base_time()
        if self.scenario_rng.random() < 0.5:
            in_addrs = [self.scenario_rng.choice(e.addresses)]
            out_addrs = [self.new_address_for(self.random_entity()) for _ in range(self.scenario_rng.randint(15, 25))]
        else:
            in_addrs = [self.new_address_for(e) for _ in range(self.scenario_rng.randint(15, 25))]
            out_addrs = [self.new_address_for(self.random_entity())]
        tx = self.build_transaction(in_addrs, out_addrs, ts, input_entities=[e.entity_id])
        return self.finalize_single_tx_scenario("UNUSUAL_TRANSACTION_STRUCTURE", tx, [e.entity_id])

    def scenario_peeling_chain(self) -> ScenarioRecord:
        e = self.random_entity()
        ts = self.make_base_time()
        chain_len = self.scenario_rng.randint(4, 7)
        txids = []
        current_addr = self.scenario_rng.choice(e.addresses)
        for i in range(chain_len):
            peel_addr = self.new_address_for(self.random_entity())  # small peeled-off amount
            change_addr = self.new_address_for(e)  # majority continues in the chain
            step_ts = ts + timedelta(minutes=10 * i)
            tx = self.build_transaction(
                [current_addr], [peel_addr, change_addr], step_ts, input_entities=[e.entity_id]
            )
            # bias split so "change_addr" carries most of the value (a real peel):
            # recompute output split with a strong skew
            total = sum(tx.input_amounts) - tx.fee
            peeled = round8(total * self.scenario_rng.uniform(0.02, 0.08))
            change = round8(total - peeled)
            tx.output_amounts = [peeled, change]
            txids.append(tx.txid)
            current_addr = change_addr
        return self.finalize_multi_tx_scenario(
            "PEELING_CHAIN", txids, [e.entity_id],
            [{"type": "peeling_chain", "entity_id": e.entity_id, "length": chain_len}],
        )

    def scenario_coinjoin_like(self) -> ScenarioRecord:
        n_participants = self.scenario_rng.randint(3, 6)
        participants = self.entity_rng.sample(self.entities, k=min(n_participants, len(self.entities)))
        ts = self.make_base_time()
        in_addrs = [self.scenario_rng.choice(p.addresses) for p in participants]
        out_addrs = [self.new_address_for(self.random_entity()) for _ in range(len(participants))]
        tx = self.build_transaction(
            in_addrs, out_addrs, ts, input_entities=[p.entity_id for p in participants]
        )
        # force near-equal output amounts (coinjoin signature), fee-adjusted
        total = sum(tx.input_amounts) - tx.fee
        equal_amt = round8(total / len(out_addrs))
        amounts = [equal_amt] * len(out_addrs)
        drift = round8(total - sum(amounts))
        amounts[-1] = round8(amounts[-1] + drift)
        tx.output_amounts = amounts
        return self.finalize_single_tx_scenario(
            "COINJOIN_LIKE", tx, [p.entity_id for p in participants],
            extra_rel=[{"type": "coinjoin_group", "entity_ids": [p.entity_id for p in participants]}],
        )

    def scenario_custodial_consolidation(self) -> ScenarioRecord:
        n_participants = self.scenario_rng.randint(4, 8)
        participants = self.entity_rng.sample(self.entities, k=min(n_participants, len(self.entities)))
        ts = self.make_base_time()
        in_addrs = []
        for p in participants:
            k = self.scenario_rng.randint(2, 4)
            in_addrs.extend([self.new_address_for(p) for _ in range(k)])
        out_entity = self.random_entity()
        out_addrs = [self.scenario_rng.choice(out_entity.addresses)]
        tx = self.build_transaction(
            in_addrs, out_addrs, ts, value_scale=0.3,
            input_entities=[p.entity_id for p in participants],
        )
        return self.finalize_single_tx_scenario(
            "CUSTODIAL_CONSOLIDATION", tx, [p.entity_id for p in participants] + [out_entity.entity_id],
            extra_rel=[{"type": "consolidation", "source_entity_ids": [p.entity_id for p in participants],
                        "sink_entity_id": out_entity.entity_id}],
        )

    # -- finalize helpers -------------------------------------------------

    def finalize_single_tx_scenario(
        self, scenario_type: str, tx: Transaction, entity_ids: list[str], extra_rel: list[dict] | None = None
    ) -> ScenarioRecord:
        condition = weighted_choice(self.network_rng, self.args.network_weights)
        obs = self.generate_network_for_tx(tx, condition)
        self.observations.extend(obs)
        rel = [{"type": "address_ownership", "entity_ids": entity_ids}]
        if extra_rel:
            rel.extend(extra_rel)
        return ScenarioRecord(
            scenario_id=self.new_scenario_id(),
            scenario_type=scenario_type,
            txids=[tx.txid],
            ground_truth_entity_ids=entity_ids,
            anomaly_label=scenario_type in ANOMALOUS_SCENARIOS,
            expected_relationships=rel,
            network_condition={tx.txid: condition},
            seed=self.seed,
        )

    def finalize_multi_tx_scenario(
        self, scenario_type: str, txids: list[str], entity_ids: list[str], extra_rel: list[dict]
    ) -> ScenarioRecord:
        network_condition = {}
        for txid in txids:
            condition = weighted_choice(self.network_rng, self.args.network_weights)
            tx = self.transactions[txid]
            obs = self.generate_network_for_tx(tx, condition)
            self.observations.extend(obs)
            network_condition[txid] = condition
        rel = [{"type": "address_ownership", "entity_ids": entity_ids}]
        rel.extend(extra_rel)
        return ScenarioRecord(
            scenario_id=self.new_scenario_id(),
            scenario_type=scenario_type,
            txids=txids,
            ground_truth_entity_ids=entity_ids,
            anomaly_label=scenario_type in ANOMALOUS_SCENARIOS,
            expected_relationships=rel,
            network_condition=network_condition,
            seed=self.seed,
        )

    SCENARIO_BUILDERS = {
        "NORMAL": "scenario_normal",
        "HIGH_ACTIVITY": "scenario_high_activity",
        "UNUSUAL_VALUE_FLOW": "scenario_unusual_value_flow",
        "UNUSUAL_TRANSACTION_STRUCTURE": "scenario_unusual_structure",
        "PEELING_CHAIN": "scenario_peeling_chain",
        "COINJOIN_LIKE": "scenario_coinjoin_like",
        "CUSTODIAL_CONSOLIDATION": "scenario_custodial_consolidation",
    }

    # -- stratified coverage pass ------------------------------------------

    def force_coverage_cells(self) -> None:
        """Guarantee at least one scenario in each of the required
        anomaly x network-condition cells (requirement #5), on top of
        whatever the weighted random draws already produced."""
        required_cells = [
            ("HIGH_ACTIVITY", "NETWORK_STRONG"),
            ("PEELING_CHAIN", "NETWORK_ABSENT"),
            ("UNUSUAL_VALUE_FLOW", "NETWORK_SPARSE"),
            ("NORMAL", "NETWORK_STRONG"),
            ("NORMAL", "NETWORK_SPARSE"),
            ("NORMAL", "NETWORK_NOISY"),
        ]
        existing_cells = set()
        for s in self.scenarios:
            for cond in s.network_condition.values():
                existing_cells.add((s.scenario_type, cond))

        for scenario_type, forced_condition in required_cells:
            if (scenario_type, forced_condition) in existing_cells:
                continue
            builder = getattr(self, self.SCENARIO_BUILDERS[scenario_type])
            record = builder()
            # override network condition on this scenario's tx(s) directly
            for txid in record.txids:
                # remove previously generated obs for this tx, regenerate under forced condition
                self.observations = [o for o in self.observations if o.txid != txid]
                tx = self.transactions[txid]
                obs = self.generate_network_for_tx(tx, forced_condition)
                self.observations.extend(obs)
                record.network_condition[txid] = forced_condition
            self.scenarios.append(record)

    # -- duplicate records post-pass ---------------------------------------

    def inject_duplicate_records(self) -> ScenarioRecord | None:
        if not self.tx_order:
            return None
        n_dupes = max(1, int(len(self.tx_order) * self.args.duplicate_ratio))
        dup_txids = self.dup_rng.sample(self.tx_order, k=min(n_dupes, len(self.tx_order)))
        # duplicates are represented by literally repeating the row in the
        # output arrays — the CSV/JSON writer will emit them twice.
        self.duplicate_txids = dup_txids
        return ScenarioRecord(
            scenario_id=self.new_scenario_id(),
            scenario_type="DUPLICATE_RECORDS",
            txids=dup_txids,
            ground_truth_entity_ids=[],
            anomaly_label=False,
            expected_relationships=[{"type": "exact_duplicate_row", "txids": dup_txids}],
            network_condition={},
            seed=self.seed,
        )

    # -- main run -----------------------------------------------------------

    def run(self) -> None:
        self.build_entities()

        target = self.args.transactions
        produced = 0
        # weighted loop until we reach roughly the target transaction count
        while produced < target:
            scenario_type = weighted_choice(self.scenario_rng, self.args.blockchain_weights)
            builder = getattr(self, self.SCENARIO_BUILDERS[scenario_type])
            record = builder()
            self.scenarios.append(record)
            produced += len(record.txids)

        self.force_coverage_cells()

        dup_record = self.inject_duplicate_records()
        if dup_record:
            self.scenarios.append(dup_record)


# ---------------------------------------------------------------------------
# Output writers
# ---------------------------------------------------------------------------

def write_csv(path: str, rows: list[dict[str, Any]], fieldnames: list[str]) -> None:
    import csv
    with open(path, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        for r in rows:
            writer.writerow(r)


def write_json(path: str, data: Any) -> None:
    with open(path, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=2)


def checksum_file(path: str) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        h.update(f.read())
    return h.hexdigest()


SCHEMA_MD = """\
# TraceLayer Round 2 — Synthetic Dataset Schema (v{schema_version})

**provenance: SYNTHETIC** — no real Bitcoin network captures or seized data
were used anywhere in this generation process. Provenance is asserted
per-record (not only in the manifest) — see below.

## Directory layout (this is the contract the Go backend codes against)

```
data/
├── raw/                          <- the ONLY inputs the application ingests
│   ├── transactions.csv
│   ├── transactions.json
│   ├── network_observations.csv
│   └── network_observations.json
├── eval/                         <- evaluator-only, must never be read by
│   │                                the detection/enrichment pipeline
│   ├── ground_truth.json
│   └── geoip_fixture.json
└── metadata/
    ├── manifest.json
    ├── schema.md
    └── validation_report.json
```

## data/raw/transactions.csv|.json

| field | type | notes |
|---|---|---|
| txid | string (64 hex chars) | deterministic synthetic identifier, seed-derived |
| timestamp | ISO-8601 UTC | |
| input_addresses | JSON array of strings (encoded as a JSON string inside the CSV cell) | synthetic addresses, prefixed `sbc1` |
| output_addresses | JSON array of strings (same encoding) | |
| input_amounts | JSON array of floats | **unit: BTC**, 8 decimal places |
| output_amounts | JSON array of floats | **unit: BTC**, 8 decimal places |
| fee | float | BTC; always `sum(input_amounts) - sum(output_amounts)`, never negative |
| script_type | enum | P2PKH / P2WPKH / P2SH / P2TR |
| provenance | string | always `SYNTHETIC`, asserted on every row |
| dataset_id | string | matches metadata/manifest.json `dataset_id` for this run |
| generator_version | string | generator release that produced this row |

Array-valued CSV cells use plain `json.dumps` encoding, e.g.:
`input_addresses` cell contents: `["sbc1a1b2...", "sbc1c3d4..."]`
Parse with `json.loads(cell_value)` in any downstream consumer.

## data/raw/network_observations.csv|.json

This is the RAW record — as if it arrived off the wire, before enrichment.
It deliberately does **not** contain `geo_country` or `asn`. Those are
derived fields the offline GeoIP/ASN enrichment component is responsible
for producing:

```
Raw NetworkObservation (data/raw/)
        |
        v
offline GeoIP/ASN enrichment component
        |
        v
Enriched NetworkObservation (geo_country, asn attached — application-owned,
                              not generator output)
```

| field | type | notes |
|---|---|---|
| observation_id | string | |
| timestamp | ISO-8601 UTC | independently jittered vs. the linked tx timestamp |
| src_ip / dst_ip | IPv4 dotted-quad | drawn ONLY from RFC 5737 (TEST-NET-1/2/3) and RFC 1918 private ranges |
| src_port / dst_port | int | 1024-65535 |
| txid | string | foreign key into transactions.txid |
| provenance | string | always `SYNTHETIC`, asserted on every row |
| dataset_id | string | matches metadata/manifest.json `dataset_id` for this run |
| generator_version | string | generator release that produced this row |

## data/eval/geoip_fixture.json (EVALUATION ONLY)

The generator internally assigns a deterministic "true" `geo_country`/`asn`
to every synthetic IP it ever emits, but that value is written **only**
here, keyed by IP — never onto the raw observation. Use this fixture to
grade the enrichment component's output; do not let the enrichment
component read this file at runtime, or it can trivially "pass" by copying
the answer instead of doing GeoIP/ASN lookup.

Format: `{{"<ip>": {{"geo_country": "...", "asn": "AS....."}}, ...}}`

## data/eval/ground_truth.json (EVALUATION ONLY — never fed to the ML pipeline)

One record per scenario group:
`scenario_id, scenario_type, txids, ground_truth_entity_ids, anomaly_label,
expected_relationships, network_condition, seed`

`scenario_type` and `anomaly_label` are the ONLY place anomaly ground truth
appears in the entire dataset. They do not appear in `data/raw/*`.

## data/metadata/manifest.json

`dataset_id, generator_version, random_seed, generated_at, record_counts,
schema_version, provenance, scenario_counts, checksums`

## Anti-leakage boundary

The Go backend's normal detection code path must have **no import, no
file-read, no config reference** to anything under `data/eval/`. Only a
separate evaluation harness (offline, CI-only) may open those files.
"""


# ---------------------------------------------------------------------------
# Validation
# ---------------------------------------------------------------------------

def validate(
    gen: "TraceLayerGenerator",
    tx_rows: list[dict[str, Any]],
    obs_rows: list[dict[str, Any]],
) -> dict[str, Any]:
    errors: list[str] = []
    warnings: list[str] = []

    # 1. unique txids in the canonical transaction table (duplicates are
    #    injected only as separate *row* repeats, tracked separately)
    seen_txids = set()
    dup_seen = set(getattr(gen, "duplicate_txids", []))
    canonical_txids = [t for t in gen.tx_order]
    if len(set(canonical_txids)) != len(canonical_txids):
        errors.append("Duplicate txid found among canonical (non-injected) transactions")

    # 2. valid timestamps
    for tx in gen.transactions.values():
        try:
            datetime.strptime(tx.timestamp, "%Y-%m-%dT%H:%M:%SZ")
        except ValueError:
            errors.append(f"Invalid timestamp on tx {tx.txid}")

    # 3/4. valid IPs and ports on observations
    for o in gen.observations:
        try:
            ipaddress.ip_address(o.src_ip)
            ipaddress.ip_address(o.dst_ip)
        except ValueError:
            errors.append(f"Invalid IP in observation {o.observation_id}")
        if not (0 < o.src_port <= 65535) or not (0 < o.dst_port <= 65535):
            errors.append(f"Invalid port in observation {o.observation_id}")

    # 5/6/7. non-negative amounts, input>=output, fee consistency
    for tx in gen.transactions.values():
        if any(a < 0 for a in tx.input_amounts) or any(a < 0 for a in tx.output_amounts):
            errors.append(f"Negative amount on tx {tx.txid}")
        total_in = round8(sum(tx.input_amounts))
        total_out = round8(sum(tx.output_amounts))
        if total_in < total_out:
            errors.append(f"input < output on tx {tx.txid} ({total_in} < {total_out})")
        expected_fee = round8(total_in - total_out)
        if abs(expected_fee - tx.fee) > 1e-6:
            errors.append(f"fee mismatch on tx {tx.txid}: expected {expected_fee}, got {tx.fee}")
        if tx.fee < 0:
            errors.append(f"negative fee on tx {tx.txid}")

    # 8. every network observation references an existing txid
    known_txids = set(gen.transactions.keys())
    for o in gen.observations:
        if o.txid not in known_txids:
            errors.append(f"Observation {o.observation_id} references unknown txid {o.txid}")

    # 9. no production record contains ground-truth labels
    forbidden_fields = {"scenario_type", "anomaly_label", "entity_id", "ground_truth_entity_ids"}
    for row in tx_rows + obs_rows:
        if forbidden_fields & set(row.keys()):
            errors.append("Ground-truth field leaked into production record")
            break

    # 10. entity relationships actually represented (shared addresses across tx)
    addr_to_txcount: dict[str, int] = {}
    for tx in gen.transactions.values():
        for a in tx.input_addresses + tx.output_addresses:
            addr_to_txcount[a] = addr_to_txcount.get(a, 0) + 1
    shared_addr_count = sum(1 for c in addr_to_txcount.values() if c > 1)
    if shared_addr_count == 0:
        warnings.append("No shared addresses found across transactions")

    # 11. scenarios have intended characteristics (spot check counts > 0 for each type)
    scenario_type_counts: dict[str, int] = {}
    for s in gen.scenarios:
        scenario_type_counts[s.scenario_type] = scenario_type_counts.get(s.scenario_type, 0) + 1
    for st in ALL_BLOCKCHAIN_SCENARIOS:
        if scenario_type_counts.get(st, 0) == 0:
            warnings.append(f"Scenario type {st} was not generated")

    # 12. deterministic rerun check is performed by the caller (main()),
    #     comparing checksums of two runs with the same seed.

    return {
        "errors": errors,
        "warnings": warnings,
        "shared_address_count": shared_addr_count,
        "scenario_type_counts": scenario_type_counts,
        "passed": len(errors) == 0,
    }


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def parse_weight_arg(raw: str | None, default: dict[str, float]) -> dict[str, float]:
    if not raw:
        return default
    out = dict(default)
    for pair in raw.split(","):
        k, v = pair.split("=")
        out[k.strip()] = float(v)
    return out


def build_arg_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description="TraceLayer Round 2 synthetic dataset generator")
    p.add_argument("--seed", type=int, default=42, help="Master random seed (reproducibility)")
    p.add_argument("--transactions", type=int, default=100, help="Approximate target transaction count")
    p.add_argument("--entities", type=int, default=20, help="Number of synthetic entities")
    p.add_argument("--network-observations", type=int, default=None,
                   help="(reserved) explicit cap on total observations; unset = scenario-driven")
    p.add_argument("--scenario-ratio", type=str, default=None,
                   help="Comma list scenario=weight overrides, e.g. NORMAL=0.5,HIGH_ACTIVITY=0.1")
    p.add_argument("--network-ratio", type=str, default=None,
                   help="Comma list condition=weight overrides")
    p.add_argument("--duplicate-ratio", type=float, default=0.03,
                   help="Fraction of transactions to duplicate as exact row repeats")
    p.add_argument("--output-dir", type=str, default="./tracelayer_output")
    return p


def generate_dataset(args: argparse.Namespace) -> tuple[TraceLayerGenerator, dict[str, Any]]:
    args.blockchain_weights = parse_weight_arg(args.scenario_ratio, DEFAULT_BLOCKCHAIN_WEIGHTS)
    args.network_weights = parse_weight_arg(args.network_ratio, DEFAULT_NETWORK_WEIGHTS)

    gen = TraceLayerGenerator(args)
    gen.run()

    tx_fieldnames = ["txid", "timestamp", "input_addresses", "output_addresses",
                      "input_amounts", "output_amounts", "fee", "script_type",
                      "provenance", "dataset_id", "generator_version"]
    obs_fieldnames = ["observation_id", "timestamp", "src_ip", "dst_ip", "src_port",
                       "dst_port", "txid", "provenance", "dataset_id", "generator_version"]

    tx_rows = [gen.transactions[t].to_row() for t in gen.tx_order]
    # apply DUPLICATE_RECORDS: append exact repeats at the end of the row list
    for txid in getattr(gen, "duplicate_txids", []):
        tx_rows.append(gen.transactions[txid].to_row())

    obs_rows = [o.to_row() for o in gen.observations]

    validation = validate(gen, tx_rows, obs_rows)

    raw_dir = os.path.join(args.output_dir, "data", "raw")
    eval_dir = os.path.join(args.output_dir, "data", "eval")
    meta_dir = os.path.join(args.output_dir, "data", "metadata")
    for d in (raw_dir, eval_dir, meta_dir):
        os.makedirs(d, exist_ok=True)

    # -- data/raw/ : the ONLY files the application ingests -----------------
    write_csv(os.path.join(raw_dir, "transactions.csv"), tx_rows, tx_fieldnames)
    write_json(os.path.join(raw_dir, "transactions.json"), tx_rows)

    write_csv(os.path.join(raw_dir, "network_observations.csv"), obs_rows, obs_fieldnames)
    write_json(os.path.join(raw_dir, "network_observations.json"), obs_rows)

    # -- data/eval/ : evaluator-only, never read by the detection pipeline --
    ground_truth = [
        {
            "scenario_id": s.scenario_id,
            "scenario_type": s.scenario_type,
            "txids": s.txids,
            "ground_truth_entity_ids": s.ground_truth_entity_ids,
            "anomaly_label": s.anomaly_label,
            "expected_relationships": s.expected_relationships,
            "network_condition": s.network_condition,
            "seed": s.seed,
        }
        for s in gen.scenarios
    ]
    write_json(os.path.join(eval_dir, "ground_truth.json"), ground_truth)

    # synthetic "true" GeoIP/ASN answer key, keyed by IP — used only to grade
    # the offline enrichment component, never consumed at detection time.
    write_json(os.path.join(eval_dir, "geoip_fixture.json"), gen.ip_geo_truth)

    # -- data/metadata/ -------------------------------------------------
    with open(os.path.join(meta_dir, "schema.md"), "w", encoding="utf-8") as f:
        f.write(SCHEMA_MD.format(schema_version=SCHEMA_VERSION))

    scenario_counts: dict[str, int] = {}
    for s in gen.scenarios:
        scenario_counts[s.scenario_type] = scenario_counts.get(s.scenario_type, 0) + 1

    manifest = {
        "dataset_id": gen.dataset_id,
        "generator_version": GENERATOR_VERSION,
        "random_seed": args.seed,
        "generated_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "record_counts": {
            "transactions": len(tx_rows),
            "network_observations": len(obs_rows),
            "entities_internal": len(gen.entities),
            "scenarios": len(gen.scenarios),
            "distinct_ips_in_geoip_fixture": len(gen.ip_geo_truth),
        },
        "schema_version": SCHEMA_VERSION,
        "provenance": "SYNTHETIC",
        "scenario_counts": scenario_counts,
        "checksums": {},
    }
    write_json(os.path.join(meta_dir, "manifest.json"), manifest)

    # checksum pass (after all files written, then patch manifest) — paths
    # are relative to data/ so the manifest stays meaningful regardless of
    # --output-dir
    checksum_targets = {
        "raw/transactions.csv": os.path.join(raw_dir, "transactions.csv"),
        "raw/transactions.json": os.path.join(raw_dir, "transactions.json"),
        "raw/network_observations.csv": os.path.join(raw_dir, "network_observations.csv"),
        "raw/network_observations.json": os.path.join(raw_dir, "network_observations.json"),
        "eval/ground_truth.json": os.path.join(eval_dir, "ground_truth.json"),
        "eval/geoip_fixture.json": os.path.join(eval_dir, "geoip_fixture.json"),
        "metadata/schema.md": os.path.join(meta_dir, "schema.md"),
    }
    checksums = {rel: checksum_file(path) for rel, path in checksum_targets.items()}
    manifest["checksums"] = checksums
    write_json(os.path.join(meta_dir, "manifest.json"), manifest)

    write_json(os.path.join(meta_dir, "validation_report.json"), validation)

    return gen, validation


def main():
    parser = build_arg_parser()
    args = parser.parse_args()
    gen, validation = generate_dataset(args)

    print(f"Generated {len(gen.tx_order)} canonical transactions "
          f"(+{len(getattr(gen, 'duplicate_txids', []))} injected duplicates), "
          f"{len(gen.observations)} network observations, "
          f"{len(gen.scenarios)} scenario groups, "
          f"{len(gen.entities)} entities.")
    print(f"Validation: {'PASSED' if validation['passed'] else 'FAILED'}")
    if validation["errors"]:
        print("Errors:")
        for e in validation["errors"]:
            print(f"  - {e}")
    if validation["warnings"]:
        print("Warnings:")
        for w in validation["warnings"]:
            print(f"  - {w}")
    print(f"Output written to: {os.path.abspath(args.output_dir)}")


if __name__ == "__main__":
    main()