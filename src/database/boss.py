from src.database.db_handler import get_connection

def get_user_stage(user_id: int) -> int:
    conn = get_connection()
    try:
        row = conn.execute("SELECT boss_league_stage FROM users WHERE user_id = ?", (user_id,)).fetchone()
        return row["boss_league_stage"] if row else 0
    finally:
        conn.close()

def set_user_stage(user_id: int, stage: int):
    conn = get_connection()
    try:
        conn.execute("UPDATE users SET boss_league_stage = ? WHERE user_id = ?", (stage, user_id))
        conn.commit()
    finally:
        conn.close()
