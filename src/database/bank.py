from src.database.db_handler import get_connection


def ensure_bank_column():
    conn = get_connection()
    try:
        conn.execute("ALTER TABLE users ADD COLUMN bank INTEGER DEFAULT 0")
        conn.commit()
    except Exception:
        pass
    finally:
        conn.close()


def get_bank_data(user_id):
    conn = get_connection()
    row = conn.execute("SELECT balance, bank FROM users WHERE user_id = ?", (user_id,)).fetchone()
    conn.close()
    if row:
        return row['balance'], row['bank']
    return 100, 0


def deposit_money(user_id, amount):
    conn = get_connection()
    try:
        row = conn.execute("SELECT balance, bank FROM users WHERE user_id = ?", (user_id,)).fetchone()
        if not row: return "ERROR"

        wallet, bank = row['balance'], row['bank']
        MAX_BANK = 500

        if bank >= MAX_BANK:
            return "BANK_FULL"
        space_left = MAX_BANK - bank
        actual_deposit = min(amount, space_left)
        if wallet < actual_deposit:
            return "NO_MONEY"
        conn.execute("UPDATE users SET balance = balance - ?, bank = bank + ? WHERE user_id = ?",
                     (actual_deposit, actual_deposit, user_id))
        conn.commit()
        return "SUCCESS"
    finally:
        conn.close()


def withdraw_money(user_id, amount) -> bool:
    conn = get_connection()
    try:
        row = conn.execute("SELECT bank FROM users WHERE user_id = ?", (user_id,)).fetchone()
        if not row or row['bank'] < amount:
            return False
        conn.execute("UPDATE users SET balance = balance + ?, bank = bank - ? WHERE user_id = ?",
                     (amount, amount, user_id))
        conn.commit()
        return True
    finally:
        conn.close()
