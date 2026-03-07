from src.database.db_handler import get_connection


def add_money_to_all(amount):
    conn = get_connection()
    try:
        cursor = conn.execute("UPDATE users SET balance = balance + ?", (amount,))
        rows_affected = cursor.rowcount
        conn.commit()
        return rows_affected
    except Exception as e:
        print(f"Erreur Airdrop: {e}")
        return 0
    finally:
        conn.close()


def get_top_users(limit=5):
    conn = get_connection()
    rows = conn.execute("SELECT user_id, balance FROM users ORDER BY balance DESC LIMIT ?", (limit,)).fetchall()
    conn.close()
    for row in rows:
        yield {"user_id": row[0], "balance": row[1]}


def pay_random_broke_user(amount, max_balance=0):
    """Sélectionne UN utilisateur fauché au hasard et le renfloue."""
    conn = get_connection()
    try:
        cursor = conn.execute(
            "SELECT user_id FROM users WHERE balance + bank <= ? ORDER BY RANDOM() LIMIT 1",
            (max_balance,)
        )
        row = cursor.fetchone()

        if not row:
            return None

        winner_id = row['user_id']

        conn.execute("UPDATE users SET balance = balance + ? WHERE user_id = ?", (amount, winner_id))
        conn.commit()

        return winner_id
    except Exception as e:
        print(f"Erreur Loterie Misère: {e}")
        return None
    finally:
        conn.close()


def reset_user_limit(user_id, activity_name):
    conn = get_connection()
    conn.execute("UPDATE game_limits SET count = 0 WHERE user_id = ? AND game_name = ?",
                 (user_id, activity_name))
    conn.commit()
    conn.close()


def get_top_glory_users(limit=5):
    from src.models.Achievement import Achievement
    conn = get_connection()
    try:
        rows = conn.execute("SELECT user_id, achievement_id FROM user_achievements").fetchall()
    finally:
        conn.close()
    
    user_glory = {}
    for row in rows:
        user_id = row["user_id"]
        ach_id = row["achievement_id"]
        ach = Achievement.get(ach_id)
        if ach:
            if user_id not in user_glory:
                user_glory[user_id] = 0
            user_glory[user_id] += ach.glory

    sorted_users = sorted(user_glory.items(), key=lambda x: x[1], reverse=True)
    return [{"user_id": uid, "glory": glory} for uid, glory in sorted_users[:limit]]


def get_top_pets(limit=5):
    conn = get_connection()
    try:
        rows = conn.execute("SELECT id, user_id, nickname, pet_type, elo FROM user_pets ORDER BY elo DESC LIMIT ?", (limit,)).fetchall()
        return [{"pet_id": row["id"], "user_id": row["user_id"], "nickname": row["nickname"], "pet_type": row["pet_type"], "elo": row["elo"]} for row in rows]
    finally:
        conn.close()

