from datetime import datetime

from src.database.db_handler import get_connection
from src.globals import BASE_JACKPOT


def get_lotto_state(server_id: int):
    conn = get_connection()
    row = conn.execute("SELECT * FROM server_lotto_state WHERE server_id = ?", (server_id,)).fetchone()
    if not row:
        import random
        winning_number = random.randint(1, 100)
        jackpot = 500
        conn.execute("INSERT INTO server_lotto_state (server_id, winning_number, jackpot) VALUES (?, ?, ?)",
                     (server_id, winning_number, jackpot))
        conn.commit()
        conn.close()
        return {"winning_number": winning_number, "jackpot": jackpot}

    conn.close()
    return {"winning_number": row['winning_number'], "jackpot": row['jackpot']}


def increment_lotto_jackpot(server_id: int, amount: int):
    conn = get_connection()
    conn.execute("UPDATE server_lotto_state SET jackpot = jackpot + ? WHERE server_id = ?", (amount, server_id))
    conn.commit()
    conn.close()


def reset_lotto(server_id: int):
    import random
    new_number = random.randint(1, 100)
    conn = get_connection()
    conn.execute("UPDATE server_lotto_state SET winning_number = ?, jackpot = ? WHERE server_id = ?", (new_number, BASE_JACKPOT, server_id))
    conn.commit()
    conn.close()
    return new_number, BASE_JACKPOT


def try_daily_lotto_bonus(server_id: int, amount: int):
    conn = get_connection()
    today_str = datetime.now().strftime("%Y-%m-%d")
    try:
        row = conn.execute("SELECT last_bonus_date FROM server_lotto_state WHERE server_id = ?", (server_id,)).fetchone()
        if not row:
            get_lotto_state(server_id)
            row = conn.execute("SELECT last_bonus_date FROM server_lotto_state WHERE server_id = ?", (server_id,)).fetchone()
            
        if not row or row['last_bonus_date'] != today_str:
            conn.execute("""
                         UPDATE server_lotto_state
                         SET jackpot         = jackpot + ?,
                             last_bonus_date = ?
                         WHERE server_id = ?
                         """, (amount, today_str, server_id))
            conn.commit()
            return True
        return False
    except Exception as e:
        print(f"Erreur Bonus Loto: {e}")
        return False
    finally:
        conn.close()
