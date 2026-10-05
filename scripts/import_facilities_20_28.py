#!/usr/bin/env python3
"""Import facilities 20 & 28 + related partner users into fac_identity."""
from __future__ import annotations

import json
import re
import subprocess
from pathlib import Path

DUMP = Path(__file__).resolve().parents[1] / "u486535318_facinect_pro(2).sql"
ALLOWED = {20, 28}

FACILITIES = [
    # (id, name, slug, active)
    (20, "strolllabstesting", "strolllabstesting", "active"),
    (28, "Drops N Smashes", "drops-n-smashes", "active"),
]

# Parsed from dump facilities_users — only users with access to 20 and/or 28.
USERS = [
    {
        "email": "rajabharathi453@gmail.com",
        "name": "Developer",
        "role": "admin",
        "page_access": [
            "dashboard", "bookings", "view_bookings", "students", "members",
            "payments", "tournaments", "enquiry", "offers", "administration",
        ],
        "facilities": [20, 28],
        "phone": "917708054914",
        "password": "$2y$10$e.JhNEo/6.bgNnsvJ4otbeOoRoTIRmI17vumMlI/ZU5I7Rcc0bFDO",
    },
    {
        "email": "sama.a.p.49@gmail.com",
        "name": "Monesh",
        "role": "sub_admin",
        "page_access": ["students"],
        "facilities": [20, 28],
        "phone": "917550193398",
        "password": None,
    },
    {
        "email": "bal.jop@gmail.com",
        "name": "Balaji",
        "role": "coach",
        "page_access": ["students", "tournaments"],
        "facilities": [28, 20],
        "phone": "919445812511",
        "password": "$2y$10$JM.G08MM2d0JGsk0jfr2Uu4YYlaXuTvUG1YuImLJKDJ3X3asBTbpK",
    },
    {
        "email": "gopilin18@gmail.com",
        "name": "Gopi",
        "role": "coach",
        "page_access": None,
        "facilities": [28],
        "phone": "919791687924",
        "password": None,
    },
    {
        "email": "bhavadharini568@gmail.com",
        "name": "Bhavadharini",
        "role": "coach",
        "page_access": ["students"],
        "facilities": [28, 20],
        "phone": "919445563654",
        "password": None,
    },
    {
        "email": "strollbyktest@gmail.com",
        "name": "TestUser",
        "role": "headcoach",
        "page_access": None,
        "facilities": [20],
        "phone": "917708054914",
        "password": None,
    },
    {
        "email": "moneshraju@gmail.com",
        "name": "Monesh",
        "role": "admin",
        "page_access": [
            "dashboard", "bookings", "view_bookings", "students", "members",
            "payments", "tournaments", "enquiry", "offers", "administration",
        ],
        "facilities": [28, 20],
        "phone": "917550193398",
        "password": None,
    },
    {
        "email": "dhanaseelanb@augmalabs.com",
        "name": "Dhanaseelan",
        "role": "admin",
        "page_access": None,
        "facilities": [20],
        "phone": "9179598054914",
        # argon2 — password login unsupported by identity bcrypt verifier; keep NULL
        "password": None,
    },
    {
        "email": "bdhanaseelan18@gmail.com",
        "name": "Dhanaseelan",
        "role": "admin",
        "page_access": None,
        "facilities": [20],
        "phone": "7305694860",
        "password": None,
    },
    {
        "email": "bdhanaseelan13@gmail.com",
        "name": "Dhanaseelan B",
        "role": "admin",
        "page_access": None,
        "facilities": [20],
        "phone": "7305694862",
        "password": None,
    },
    {
        "email": "strollbyk@gmail.com",
        "name": "Self_coach",
        "role": "coach",
        "page_access": None,
        "facilities": [28],
        "phone": "917010712515",
        "password": None,
    },
    {
        "email": "iagqmfo@outlook.com",
        "name": "Priya Raj",
        "role": "admin",
        "page_access": None,
        "facilities": [20],
        "phone": "7464365880",
        "password": None,
    },
    {
        "email": "dhanaseelanb@augmalbs.com",
        "name": "Dhanaseelan b",
        "role": "coach",
        "page_access": None,
        "facilities": [20],
        "phone": "9179598054914",
        "password": None,
    },
    {
        "email": "tsfvavd@outlook.com",
        "name": "Arjun Singh",
        "role": "admin",
        "page_access": None,
        "facilities": [20],
        "phone": "7904615227",
        "password": None,
    },
    {
        "email": "basamadh97@gmail.com",
        "name": "Balu",
        "role": "admin",
        "page_access": [
            "dashboard", "bookings", "view_bookings", "students", "members",
            "tournaments", "enquiry",
        ],
        "facilities": [28],
        "phone": "8838226071",
        "password": None,
    },
    {
        "email": "jayashreekr98@gmail.com",
        "name": "Monesh",
        "role": "coach",
        "page_access": [
            "dashboard", "bookings", "view_bookings", "students", "members",
            "payments", "tournaments", "enquiry", "offers", "administration",
        ],
        "facilities": [28, 20],
        "phone": "917550193398",
        "password": None,
    },
    {
        "email": "bal.job@gmail.com",
        "name": "Balaji",
        "role": "sub_admin",
        "page_access": [
            "dashboard", "bookings", "view_bookings", "students", "members",
            "tournaments", "enquiry",
        ],
        "facilities": [28],
        "phone": "9445812511",
        "password": None,
    },
]


def sql_str(v: str | None) -> str:
    if v is None:
        return "NULL"
    return "'" + v.replace("\\", "\\\\").replace("'", "''") + "'"


def phone_e164(raw: str | None) -> str | None:
    if not raw:
        return None
    d = re.sub(r"\D+", "", raw)
    if len(d) == 10:
        d = "91" + d
    if len(d) < 10:
        return None
    return d


def build_sql() -> str:
    lines = [
        "-- Import facilities 20 & 28 from Facinect dump (PostgreSQL)",
    ]
    for fid, name, slug, status in FACILITIES:
        lines.append(
            f"INSERT INTO facilities (id, name, slug, status) VALUES ({fid}, {sql_str(name)}, {sql_str(slug)}, {sql_str(status)}) "
            f"ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, slug=EXCLUDED.slug, status=EXCLUDED.status;"
        )
    lines.append("SELECT setval(pg_get_serial_sequence('facilities','id'), GREATEST((SELECT MAX(id) FROM facilities), 1));")

    used_phones: set[str] = set()

    for u in USERS:
        email = u["email"].lower().strip()
        name = u["name"]
        pwd = u["password"]
        if pwd and not pwd.startswith(("$2y$", "$2a$", "$2b$")):
            pwd = None
        phone = phone_e164(u.get("phone"))
        if phone and phone in used_phones:
            phone = None
        if phone:
            used_phones.add(phone)

        lines.append(
            f"INSERT INTO users (email, phone_e164, password_hash, full_name, status) "
            f"VALUES ({sql_str(email)}, {sql_str(phone)}, {sql_str(pwd)}, {sql_str(name)}, 'active') "
            f"ON CONFLICT (email) DO UPDATE SET "
            f"full_name=EXCLUDED.full_name, "
            f"password_hash=COALESCE(EXCLUDED.password_hash, users.password_hash), "
            f"status='active';"
        )

        role = (u.get("role") or "admin").strip() or "admin"
        page = u.get("page_access")
        page_sql = "NULL" if not page else sql_str(json.dumps(page, separators=(",", ":"))) + "::jsonb"

        for fid in u["facilities"]:
            if fid not in ALLOWED:
                continue
            lines.append(
                f"INSERT INTO user_facility_memberships (user_id, facility_id, role, page_access, status) "
                f"SELECT id, {fid}, {sql_str(role)}, {page_sql}, 'active' FROM users WHERE email = {sql_str(email)} "
                f"ON CONFLICT (user_id, facility_id) DO UPDATE SET "
                f"role=EXCLUDED.role, page_access=EXCLUDED.page_access, status='active';"
            )

    lines.append(
        "SELECT 'facilities' AS t, COUNT(*)::text AS n FROM facilities WHERE id IN (20,28) "
        "UNION ALL SELECT 'memberships_20_28', COUNT(*)::text FROM user_facility_memberships WHERE facility_id IN (20,28);"
    )
    return "\n".join(lines) + "\n"


def main() -> None:
    out = Path(__file__).resolve().parents[1] / "sql" / "seeds" / "facilities_20_28.sql"
    out.write_text(build_sql(), encoding="utf-8")
    print(f"wrote {out}")

    cmd = [
        "docker", "exec", "-i", "facinect-postgres",
        "psql", "-U", "identity", "-d", "fac_identity", "-v", "ON_ERROR_STOP=1",
    ]
    proc = subprocess.run(cmd, input=out.read_text(encoding="utf-8"), text=True, capture_output=True)
    print(proc.stdout)
    if proc.returncode != 0:
        print(proc.stderr)
        raise SystemExit(proc.returncode)
    print("import ok")


if __name__ == "__main__":
    main()
