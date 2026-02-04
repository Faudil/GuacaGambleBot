from src.database.db_handler import get_connection


def create_bet_db(creator_id, description, opt1, opt2):
    conn = get_connection()
    cursor = conn.cursor()
    cursor.execute("""
                   INSERT INTO bets (creator_id, description, option1, option2, status)
                   VALUES (?, ?, ?, ?, 'OPEN')
                   """, (creator_id, description, opt1, opt2))
    bet_id = cursor.lastrowid
    conn.commit()
    conn.close()
    return bet_id


def get_bet_data(bet_id):
    conn = get_connection()
    bet = conn.execute("SELECT * FROM bets WHERE id = ?", (bet_id,)).fetchone()
    if not bet:
        conn.close()
        return None
    wagers = conn.execute("SELECT * FROM wagers WHERE bet_id = ?", (bet_id,)).fetchall()
    conn.close()
    return {
        "id": bet['id'],
        "creator": bet['creator_id'],
        "description": bet['description'],
        "options": [bet['option1'], bet['option2']],
        "status": bet['status'],
        "winner": bet['winner'],
        "wagers": [{"user_id": w['user_id'], "option": w['option'], "amount": w['amount']} for w in wagers]
    }


def freeze_bet(bet_id):
    conn = get_connection()
    conn.execute("UPDATE bets SET status = 'FROZEN' WHERE id = ?", (bet_id,))
    conn.close()


def add_wager(bet_id, user_id, option, amount):
    conn = get_connection()
    conn.execute("INSERT INTO wagers (bet_id, user_id, option, amount) VALUES (?, ?, ?, ?)",
                 (bet_id, user_id, option, amount))
    conn.commit()
    conn.close()


def close_bet_db(bet_id, winner):
    conn = get_connection()
    conn.execute("UPDATE bets SET status = 'CLOSED', winner = ? WHERE id = ?", (winner, bet_id))
    conn.commit()
    conn.close()
