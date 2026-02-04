from datetime import datetime

from src.database.db_handler import get_connection
from src.globals import BASE_JACKPOT


def get_lotto_state():
    conn = get_connection()
    row = conn.execute("SELECT * FROM lotto_state WHERE id = 1").fetchone()
    if not row:
        import random
        winning_number = random.randint(1, 100)
        jackpot = 500
        conn.execute("INSERT INTO lotto_state (id, winning_number, jackpot) VALUES (1, ?, ?)",
                     (winning_number, jackpot))
        conn.commit()
        conn.close()
        return {"winning_number": winning_number, "jackpot": jackpot}

    conn.close()
    return {"winning_number": row['winning_number'], "jackpot": row['jackpot']}


def increment_lotto_jackpot(amount):
    conn = get_connection()
    conn.execute("UPDATE lotto_state SET jackpot = jackpot + ? WHERE id = 1", (amount,))
    conn.commit()
    conn.close()


def reset_lotto():
    import random
    new_number = random.randint(1, 100)
    conn = get_connection()
    conn.execute("UPDATE lotto_state SET winning_number = ?, jackpot = ? WHERE id = 1", (new_number, BASE_JACKPOT))
    conn.commit()
    conn.close()
    return new_number, BASE_JACKPOT


def try_daily_lotto_bonus(amount):
    conn = get_connection()
    today_str = datetime.now().strftime("%Y-%m-%d")
    try:
        row = conn.execute("SELECT last_bonus_date FROM lotto_state WHERE id = 1").fetchone()
        if not row or row['last_bonus_date'] != today_str:
            conn.execute("""
                         UPDATE lotto_state
                         SET jackpot         = jackpot + ?,
                             last_bonus_date = ?
                         WHERE id = 1
                         """, (amount, today_str))
            conn.commit()
            return True
        return False
    except Exception as e:
        print(f"Erreur Bonus Loto: {e}")
        return False
    finally:
        conn.close()
