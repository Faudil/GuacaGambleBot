from src.database.db_handler import get_connection

def set_announcement_channel(server_id: int, channel_id: int):
    conn = get_connection()
    try:
        conn.execute("""
            INSERT INTO server_settings (server_id, announcement_channel_id)
            VALUES (?, ?)
            ON CONFLICT(server_id) DO UPDATE SET announcement_channel_id = excluded.announcement_channel_id
        """, (server_id, channel_id))
        conn.commit()
    finally:
        conn.close()

def get_announcement_channel(server_id: int) -> int:
    conn = get_connection()
    try:
        row = conn.execute("SELECT announcement_channel_id FROM server_settings WHERE server_id = ?", (server_id,)).fetchone()
        if row:
            return row["announcement_channel_id"]
        return None
    finally:
        conn.close()

def disable_announcements(server_id: int):
    set_announcement_channel(server_id, 0)

def set_language(server_id: int, lang: str):
    conn = get_connection()
    try:
        conn.execute("""
            INSERT INTO server_settings (server_id, language)
            VALUES (?, ?)
            ON CONFLICT(server_id) DO UPDATE SET language = excluded.language
        """, (server_id, lang))
        conn.commit()
    finally:
        conn.close()

def get_language(server_id: int) -> str:
    if server_id is None:
        return 'fr'
    conn = get_connection()
    try:
        row = conn.execute("SELECT language FROM server_settings WHERE server_id = ?", (server_id,)).fetchone()
        if row and row["language"]:
            return row["language"]
        return 'fr'
    finally:
        conn.close()
