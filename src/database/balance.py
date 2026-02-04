from src.database.db_handler import get_connection
from src.globals import STARTING_BALANCE


def get_balance(user_id):
    conn = get_connection()
    row = conn.execute("SELECT balance FROM users WHERE user_id = ?", (user_id,)).fetchone()
    conn.close()
    if row:
        return row['balance']
    update_balance(user_id, 0)
    return STARTING_BALANCE


def update_balance(user_id, amount) -> int:
    conn = get_connection()
    try:
        conn.execute("""
                     INSERT INTO users (user_id, balance)
                     VALUES (?, ?) ON CONFLICT(user_id) DO
                     UPDATE SET balance = balance + ?
                     """, (user_id, STARTING_BALANCE + amount, amount))
        conn.commit()
        new_bal = conn.execute("SELECT balance FROM users WHERE user_id = ?", (user_id,)).fetchone()['balance']
        return new_bal
    finally:
        conn.close()
