from src.database.db_handler import get_connection


def get_total_debt(user_id):
    conn = get_connection()
    conn.execute("""CREATE TABLE IF NOT EXISTS loans
                    (
                        id INTEGER PRIMARY KEY,
                        borrower_id INTEGER,
                        lender_id INTEGER,
                        amount_due INTEGER,
                        created_at TEXT
                    )""")

    row = conn.execute("SELECT SUM(amount_due) as total FROM loans WHERE borrower_id = ?", (user_id,)).fetchone()
    conn.close()
    return row['total'] if row and row['total'] else 0

def get_creditors(user_id):
    conn = get_connection()
    rows = conn.execute("SELECT amount_due, lender_id FROM loans WHERE borrower_id = ?", (user_id,)).fetchall()
    conn.close()
    return rows


def create_loan(lender_id, borrower_id, amount):
    conn = get_connection()
    try:
        lender_bal = conn.execute("SELECT balance FROM users WHERE user_id = ?", (lender_id,)).fetchone()['balance']
        if lender_bal < amount:
            return "LENDER_BROKE"
        total_repay = int(amount * 1.10)
        current_debt = \
        conn.execute("SELECT SUM(amount_due) as total FROM loans WHERE borrower_id = ?", (borrower_id,)).fetchone()[
            'total'] or 0
        if (current_debt + total_repay) > 500:
            return "DEBT_LIMIT"
        conn.execute("UPDATE users SET balance = balance - ? WHERE user_id = ?", (amount, lender_id))
        conn.execute("UPDATE users SET balance = balance + ? WHERE user_id = ?", (amount, borrower_id))
        conn.execute(
            "INSERT INTO loans (borrower_id, lender_id, amount_due, created_at) VALUES (?, ?, ?, datetime('now'))",
            (borrower_id, lender_id, total_repay))
        conn.commit()
        return "SUCCESS"
    except Exception as e:
        print(e)
        return "ERROR"
    finally:
        conn.close()

def repay_debt_logic(borrower_id, payment_amount):
    conn = get_connection()
    remaining_payment = payment_amount
    refund_details = []
    try:
        loans = conn.execute("SELECT * FROM loans WHERE borrower_id = ? ORDER BY id ASC", (borrower_id,)).fetchall()
        for loan in loans:
            if remaining_payment <= 0:
                break
            to_pay = min(remaining_payment, loan['amount_due'])
            new_amount = loan['amount_due'] - to_pay
            if new_amount <= 0:
                conn.execute("DELETE FROM loans WHERE id = ?", (loan['id'],))
            else:
                conn.execute("UPDATE loans SET amount_due = ? WHERE id = ?", (new_amount, loan['id']))
            conn.execute("UPDATE users SET balance = balance + ? WHERE user_id = ?", (to_pay, loan['lender_id']))
            remaining_payment -= to_pay
            refund_details.append((loan['lender_id'], to_pay))
        conn.commit()
        return payment_amount - remaining_payment, refund_details

    finally:
        conn.close()