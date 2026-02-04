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


