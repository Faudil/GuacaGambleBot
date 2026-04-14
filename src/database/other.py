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


def get_top_users(limit=5, server_member_ids=None):
    conn = get_connection()
    if server_member_ids is not None and len(server_member_ids) > 0:
        marks = ','.join('?' * len(server_member_ids))
        query = f"SELECT user_id, balance FROM users WHERE user_id IN ({marks}) ORDER BY balance DESC LIMIT ?"
        params = tuple(server_member_ids) + (limit,)
        rows = conn.execute(query, params).fetchall()
    else:
        rows = conn.execute("SELECT user_id, balance FROM users ORDER BY balance DESC LIMIT ?", (limit,)).fetchall()
    conn.close()
    for row in rows:
        yield {"user_id": row[0], "balance": row[1]}


def pay_random_broke_user(amount, max_balance=0):
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


def get_top_glory_users(limit=5, server_member_ids=None, server_id=None):
    from src.models.Achievement import Achievement
    from src.database.pets import get_all_pet_ranks, RANK_GLORY
    conn = get_connection()
    try:
        if server_member_ids is not None and len(server_member_ids) > 0:
            marks = ','.join('?' * len(server_member_ids))
            query = f"SELECT user_id, achievement_id FROM user_achievements WHERE user_id IN ({marks})"
            rows = conn.execute(query, tuple(server_member_ids)).fetchall()
        else:
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

    ranks = get_all_pet_ranks(server_id)
    for pet_id, data in ranks.items():
        user_id = data["user_id"]
        if server_member_ids is not None and user_id not in server_member_ids:
            continue
        if user_id not in user_glory:
            user_glory[user_id] = 0
        user_glory[user_id] += RANK_GLORY.get(data["rank"], 0)

    sorted_users = sorted(user_glory.items(), key=lambda x: x[1], reverse=True)
    return [{"user_id": uid, "glory": glory} for uid, glory in sorted_users[:limit]]


def get_top_pets(limit=5, server_id=None):
    conn = get_connection()
    try:
        if server_id:
            rows = conn.execute("""
                SELECT up.id, up.user_id, up.nickname, up.pet_type, COALESCE(spe.elo, up.elo) as elo
                FROM user_pets up
                LEFT JOIN server_pet_elo spe ON up.id = spe.pet_id AND spe.server_id = ?
                ORDER BY elo DESC, up.id ASC LIMIT ?
            """, (server_id, limit)).fetchall()
        else:
            rows = conn.execute("SELECT id, user_id, nickname, pet_type, elo FROM user_pets ORDER BY elo DESC, id ASC LIMIT ?", (limit,)).fetchall()
        return [{"pet_id": row["id"], "user_id": row["user_id"], "nickname": row["nickname"], "pet_type": row["pet_type"], "elo": row["elo"]} for row in rows]
    finally:
        conn.close()

