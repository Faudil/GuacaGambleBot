import json
import os
import sqlite3
from src.data_handling import init_db, DB_FILE

JSON_FILE = 'betting_data.json'


def migrate():
    if not os.path.exists(JSON_FILE):
        print(f"❌ File {JSON_FILE} not found. Migration cancelled")
        return
    print("🚀 Starting migration JSON -> SQLite...")
    init_db()
    with open(JSON_FILE, 'r') as f:
        data = json.load(f)

    conn = sqlite3.connect(DB_FILE)
    c = conn.cursor()
    users = data.get("users", {})

    print(f"📦 Migrating {len(users)} users...")
    for user_id, balance in users.items():
        c.execute("INSERT OR REPLACE INTO users (user_id, balance) VALUES (?, ?)", (user_id, balance))

    conn.commit()
    conn.close()
    print("✅ Migration finished !")


if __name__ == "__main__":
    migrate()