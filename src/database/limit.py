from datetime import datetime

from src.database.db_handler import get_connection


def get_cooldown(user_id, activity_name):
    conn = get_connection()
    row = conn.execute("SELECT last_used FROM cooldowns WHERE user_id = ? AND activity_name = ?",
                       (user_id, activity_name)).fetchone()
    conn.close()
    if row:
        return row['last_used']
    return None


def set_cooldown(user_id, activity_name):
    now_iso = datetime.now().isoformat()
    conn = get_connection()
    conn.execute("""
                 INSERT INTO cooldowns (user_id, activity_name, last_used)
                 VALUES (?, ?, ?) ON CONFLICT(user_id, activity_name) DO
                 UPDATE SET last_used = ?
                 """, (user_id, activity_name, now_iso, now_iso))
    conn.commit()
    conn.close()


def check_game_limit(user_id, game_name, max_usage):
    conn = get_connection()
    today_str = datetime.now().strftime("%Y-%m-%d")
    row = conn.execute("SELECT date_str, count FROM game_limits WHERE user_id = ? AND game_name = ?",
                       (user_id, game_name)).fetchone()
    current_count = 0
    if row:
        if row['date_str'] != today_str:
            conn.execute("UPDATE game_limits SET date_str = ?, count = 0 WHERE user_id = ? AND game_name = ?",
                         (today_str, user_id, game_name))
            conn.commit()
            current_count = 0
        else:
            current_count = row['count']
    conn.close()
    if current_count >= max_usage:
        return False, 0
    return True, max_usage - current_count


def increment_game_limit(user_id, game_name):
    today_str = datetime.now().strftime("%Y-%m-%d")
    conn = get_connection()
    conn.execute("""
                 INSERT INTO game_limits (user_id, game_name, date_str, count)
                 VALUES (?, ?, ?, 1) ON CONFLICT(user_id, game_name) DO
                 UPDATE SET count = count + 1, date_str = ?
                 """, (user_id, game_name, today_str, today_str))
    conn.commit()
    conn.close()

