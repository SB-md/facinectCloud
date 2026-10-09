#!/usr/bin/env python3
"""Import members + students for facilities 20 & 28 from MySQL dump into fac_identity."""
from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

DUMP_CANDIDATES = [
    Path("/home/dell/Downloads/u486535318_facinect_pro(3).sql"),
    Path("/home/dell/Downloads/u486535318_facinect_pro.sql"),
    Path(__file__).resolve().parents[1] / "u486535318_facinect_pro(3).sql",
]
ALLOWED = {20, 28}


def find_dump() -> Path:
    for p in DUMP_CANDIDATES:
        if p.is_file():
            return p
    raise SystemExit("MySQL dump not found")


def extract_insert(text: str, table: str) -> str:
    m = re.search(
        rf"INSERT INTO `{re.escape(table)}`.*?VALUES\n(.*?)(?:\n\n--|\nDELIMITER|\nCREATE TABLE)",
        text,
        re.S,
    )
    if not m:
        raise SystemExit(f"INSERT for {table} not found")
    return m.group(1)


def sql_str(v: str | None) -> str:
    if v is None:
        return "NULL"
    return "'" + v.replace("'", "''") + "'"


def sql_date(v: str | None) -> str:
    if not v or v.startswith("0000"):
        return "NULL"
    return sql_str(v[:10])


def map_member_status(st: str) -> str:
    st = (st or "active").lower()
    if st in ("active", "inactive", "expired"):
        return st
    return "inactive"


def map_student_status(st: str) -> str:
    st = (st or "active").lower()
    if st in ("active", "inactive", "completed"):
        return st
    if st == "active":
        return "active"
    return "active" if st in ("active",) else "inactive"


def parse_mysql_rows(block: str) -> list[tuple]:
    """Parse simple INSERT value tuples (handles quoted strings / NULLs)."""
    rows: list[tuple] = []
    i = 0
    n = len(block)
    while i < n:
        while i < n and block[i] in " \t\r\n,":
            i += 1
        if i >= n or block[i] != "(":
            break
        i += 1
        vals: list = []
        while i < n and block[i] != ")":
            while i < n and block[i] in " \t\r\n":
                i += 1
            if i < n and block[i] == "'":
                i += 1
                buf: list[str] = []
                while i < n:
                    if block[i] == "\\" and i + 1 < n:
                        buf.append(block[i + 1])
                        i += 2
                        continue
                    if block[i] == "'":
                        if i + 1 < n and block[i + 1] == "'":
                            buf.append("'")
                            i += 2
                            continue
                        i += 1
                        break
                    buf.append(block[i])
                    i += 1
                vals.append("".join(buf))
            elif block[i : i + 4].upper() == "NULL" and (
                i + 4 >= n or block[i + 4] in ",)"
            ):
                vals.append(None)
                i += 4
            else:
                j = i
                while j < n and block[j] not in ",)":
                    j += 1
                tok = block[i:j].strip()
                if re.fullmatch(r"-?\d+", tok):
                    vals.append(int(tok))
                elif re.fullmatch(r"-?\d+\.\d+", tok):
                    vals.append(float(tok))
                else:
                    vals.append(tok)
                i = j
            while i < n and block[i] in " \t\r\n":
                i += 1
            if i < n and block[i] == ",":
                i += 1
        if i < n and block[i] == ")":
            i += 1
            rows.append(tuple(vals))
        while i < n and block[i] in " \t\r\n,;":
            i += 1
    return rows


def main() -> None:
    dump = find_dump()
    text = dump.read_text(errors="ignore")
    print(f"dump={dump}")

    plans_rows = parse_mysql_rows(extract_insert(text, "membership_plan"))
    # planId, facilityId, sportId, planName, ...
    plans = {
        int(r[0]): {
            "facility_id": int(r[1]),
            "sport_id": int(r[2]),
            "plan_name": str(r[3] or ""),
        }
        for r in plans_rows
        if int(r[1]) in ALLOWED
    }
    print(f"membership plans 20/28: {len(plans)}")

    enroll_rows = parse_mysql_rows(extract_insert(text, "membership_enrollment"))
    enrolls = [
        r
        for r in enroll_rows
        if int(r[2]) in plans  # planId
    ]
    print(f"membership enrollments: {len(enrolls)}")
    member_ids = {int(r[1]) for r in enrolls}

    members_rows = parse_mysql_rows(extract_insert(text, "members"))
    members = {int(r[0]): r for r in members_rows if int(r[0]) in member_ids}
    print(f"members referenced: {len(members)}")

    coach_plans_rows = parse_mysql_rows(extract_insert(text, "test_CoachingPlans"))
    coach_plans = {
        int(r[0]): {
            "facility_id": int(r[1]),
            "plan_name": str(r[2] or ""),
            "sport_id": int(r[10]) if r[10] is not None else None,
        }
        for r in coach_plans_rows
        if int(r[1]) in ALLOWED
    }

    stu_enroll_rows = parse_mysql_rows(extract_insert(text, "test_StudentEnrollments"))
    stu_enrolls = [r for r in stu_enroll_rows if int(r[8]) in ALLOWED]  # facilityId
    print(f"student enrollments 20/28: {len(stu_enrolls)}")

    sf_rows = parse_mysql_rows(extract_insert(text, "test_StudentFacilities"))
    sf_by_id = {int(r[0]): r for r in sf_rows if int(r[2]) in ALLOWED}
    student_ids = {int(sf_by_id[int(r[1])][1]) for r in stu_enrolls if int(r[1]) in sf_by_id}
    # also from enrollments via student_facility
    for r in stu_enrolls:
        sfid = int(r[1])
        if sfid in sf_by_id:
            student_ids.add(int(sf_by_id[sfid][1]))

    students_rows = parse_mysql_rows(extract_insert(text, "test_Students"))
    students = {int(r[0]): r for r in students_rows if int(r[0]) in student_ids}
    print(f"students referenced: {len(students)}")

    out: list[str] = [
        "-- Generated by import_members_students_20_28.py",
        "BEGIN;",
        "-- Clear demo / prior facility 20/28 memberships & enrollments (keep other facilities)",
        "DELETE FROM memberships WHERE facility_id IN (20, 28);",
        "DELETE FROM student_attendance WHERE facility_id IN (20, 28);",
        "DELETE FROM student_enrollments WHERE facility_id IN (20, 28);",
    ]

    # Members
    for mid, r in sorted(members.items()):
        name = str(r[1] or "").strip() or f"Member {mid}"
        phone = str(r[2] or "").strip() or None
        status = "active" if str(r[3] or "").lower() == "active" else "inactive"
        out.append(
            "INSERT INTO members (id, full_name, contact_phone, whatsapp, status) VALUES ("
            f"{mid}, {sql_str(name)}, {sql_str(phone) if phone else 'NULL'}, "
            f"{sql_str(phone) if phone else 'NULL'}, {sql_str(status)}) "
            "ON CONFLICT (id) DO UPDATE SET "
            "full_name=EXCLUDED.full_name, contact_phone=EXCLUDED.contact_phone, "
            "whatsapp=EXCLUDED.whatsapp, status=EXCLUDED.status, updated_at=NOW();"
        )

    for r in enrolls:
        membership_id = int(r[0])
        member_id = int(r[1])
        plan_id = int(r[2])
        team = str(r[3] or "")
        primary = bool(int(r[4] or 0))
        fee = r[6]
        start = r[7]
        end = r[8]
        status = map_member_status(str(r[9] or "active"))
        if status == "cancelled":
            status = "inactive"
        plan = plans[plan_id]
        if member_id not in members:
            continue
        fee_sql = "NULL" if fee is None else str(fee)
        out.append(
            "INSERT INTO memberships ("
            "id, facility_id, member_id, sport_id, plan_name, team_name, "
            "start_date, end_date, subscription_fee, primary_member, status"
            ") VALUES ("
            f"{membership_id}, {plan['facility_id']}, {member_id}, {plan['sport_id']}, "
            f"{sql_str(plan['plan_name'])}, {sql_str(team)}, "
            f"{sql_date(str(start) if start else None)}, {sql_date(str(end) if end else None)}, "
            f"{fee_sql}, {'TRUE' if primary else 'FALSE'}, {sql_str(status)}"
            ") ON CONFLICT (id) DO UPDATE SET "
            "facility_id=EXCLUDED.facility_id, member_id=EXCLUDED.member_id, "
            "sport_id=EXCLUDED.sport_id, plan_name=EXCLUDED.plan_name, "
            "team_name=EXCLUDED.team_name, start_date=EXCLUDED.start_date, "
            "end_date=EXCLUDED.end_date, subscription_fee=EXCLUDED.subscription_fee, "
            "primary_member=EXCLUDED.primary_member, status=EXCLUDED.status, updated_at=NOW();"
        )

    # Students
    for sid, r in sorted(students.items()):
        first = str(r[1] or "").strip() or "Student"
        last = str(r[2] or "").strip()
        email = str(r[4] or "").strip() or None
        phone = str(r[5] or "").strip() or None
        alt = str(r[6] or "").strip() or None
        gstatus = str(r[8] or "Active").lower()
        status = "active" if gstatus == "active" else "inactive"
        out.append(
            "INSERT INTO students (id, first_name, last_name, contact_email, contact_phone, whatsapp, status) VALUES ("
            f"{sid}, {sql_str(first)}, {sql_str(last)}, "
            f"{sql_str(email) if email else 'NULL'}, "
            f"{sql_str(phone) if phone else 'NULL'}, "
            f"{sql_str(alt or phone) if (alt or phone) else 'NULL'}, "
            f"{sql_str(status)}) "
            "ON CONFLICT (id) DO UPDATE SET "
            "first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name, "
            "contact_email=EXCLUDED.contact_email, contact_phone=EXCLUDED.contact_phone, "
            "whatsapp=EXCLUDED.whatsapp, status=EXCLUDED.status, updated_at=NOW();"
        )

    for r in stu_enrolls:
        eid = int(r[0])
        sfid = int(r[1])
        plan_id = int(r[2])
        start = r[4]
        end = r[5]
        fee = r[6]
        cur = str(r[7] or "Active")
        facility_id = int(r[8])
        if sfid not in sf_by_id:
            continue
        sf = sf_by_id[sfid]
        student_id = int(sf[1])
        sport_id = int(sf[3] or 0) or None
        if student_id not in students:
            continue
        plan_name = coach_plans.get(plan_id, {}).get("plan_name") or f"Plan {plan_id}"
        if sport_id is None:
            sport_id = coach_plans.get(plan_id, {}).get("sport_id")
        st = cur.lower()
        status = "active" if st == "active" else ("completed" if st == "completed" else "inactive")
        fee_sql = "NULL" if fee is None else str(fee)
        sport_sql = "NULL" if not sport_id else str(int(sport_id))
        out.append(
            "INSERT INTO student_enrollments ("
            "id, facility_id, student_id, sport_id, plan_name, start_date, end_date, actual_fee, status"
            ") VALUES ("
            f"{eid}, {facility_id}, {student_id}, {sport_sql}, {sql_str(plan_name)}, "
            f"{sql_date(str(start) if start else None)}, {sql_date(str(end) if end else None)}, "
            f"{fee_sql}, {sql_str(status)}"
            ") ON CONFLICT (id) DO UPDATE SET "
            "facility_id=EXCLUDED.facility_id, student_id=EXCLUDED.student_id, "
            "sport_id=EXCLUDED.sport_id, plan_name=EXCLUDED.plan_name, "
            "start_date=EXCLUDED.start_date, end_date=EXCLUDED.end_date, "
            "actual_fee=EXCLUDED.actual_fee, status=EXCLUDED.status, updated_at=NOW();"
        )

    out += [
        "SELECT setval(pg_get_serial_sequence('members','id'), GREATEST((SELECT MAX(id) FROM members), 1));",
        "SELECT setval(pg_get_serial_sequence('memberships','id'), GREATEST((SELECT MAX(id) FROM memberships), 1));",
        "SELECT setval(pg_get_serial_sequence('students','id'), GREATEST((SELECT MAX(id) FROM students), 1));",
        "SELECT setval(pg_get_serial_sequence('student_enrollments','id'), GREATEST((SELECT MAX(id) FROM student_enrollments), 1));",
        "SELECT 'memberships_20' AS t, COUNT(*)::text FROM memberships WHERE facility_id=20 "
        "UNION ALL SELECT 'memberships_28', COUNT(*)::text FROM memberships WHERE facility_id=28 "
        "UNION ALL SELECT 'student_enroll_20', COUNT(*)::text FROM student_enrollments WHERE facility_id=20 "
        "UNION ALL SELECT 'student_enroll_28', COUNT(*)::text FROM student_enrollments WHERE facility_id=28;",
        "COMMIT;",
    ]

    sql_path = Path(__file__).resolve().parents[1] / "sql/seeds/members_students_20_28.sql"
    sql_path.write_text("\n".join(out) + "\n", encoding="utf-8")
    print(f"wrote {sql_path} ({len(out)} statements)")

    cmd = [
        "docker",
        "exec",
        "-i",
        "-e",
        "PGPASSWORD=identity_local_change_me",
        "facinect-postgres",
        "psql",
        "-U",
        "identity",
        "-d",
        "fac_identity",
        "-v",
        "ON_ERROR_STOP=1",
    ]
    proc = subprocess.run(cmd, input="\n".join(out), text=True, capture_output=True)
    sys.stdout.write(proc.stdout)
    sys.stderr.write(proc.stderr)
    if proc.returncode != 0:
        raise SystemExit(proc.returncode)
    print("import OK")


if __name__ == "__main__":
    main()
