import sqlite3
from datetime import datetime

from src.globals import BASE_JACKPOT

DB_FILE = './data/guacabot.db'
STARTING_BALANCE = 100


def get_connection():
    conn = sqlite3.connect(DB_FILE)
    conn.row_factory = sqlite3.Row
    return conn


def init_db():
    conn = get_connection()
    c = conn.cursor()

    c.execute("""CREATE TABLE IF NOT EXISTS users
                 (
                     user_id INTEGER PRIMARY KEY,
                     balance INTEGER DEFAULT 100
                 )""")

    c.execute("""CREATE TABLE IF NOT EXISTS cooldowns
        (
            user_id INTEGER,
            activity_name TEXT,
            last_used TIMESTAMP,
            PRIMARY KEY(user_id, activity_name)
        )""")
    c.execute("""CREATE TABLE IF NOT EXISTS game_limits
    (
        user_id INTEGER,
        game_name TEXT,
        date_str TEXT,
        count INTEGER DEFAULT 0,
        PRIMARY KEY(user_id, game_name)
        )""")

    c.execute("""CREATE TABLE IF NOT EXISTS bets
                 (
                     id INTEGER PRIMARY KEY AUTOINCREMENT,
                     creator_id INTEGER,
                     description TEXT,
                     option1 TEXT,
                     option2 TEXT,
                     status VARCHAR(16) DEFAULT 'OPEN',
                     winner CHAR
                 )""")

    c.execute("""CREATE TABLE IF NOT EXISTS wagers
    (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        bet_id INTEGER,
        user_id INTEGER,
        option TEXT,
        amount INTEGER,
        FOREIGN KEY (bet_id) REFERENCES bets(id)
        )""")

    c.execute("""CREATE TABLE IF NOT EXISTS items
                 (
                     id INTEGER PRIMARY KEY AUTOINCREMENT,
                     name TEXT UNIQUE,
                     price INTEGER,
                     description TEXT,
                     effect_type TEXT
                 )""")

    c.execute("""CREATE TABLE IF NOT EXISTS inventory
    (
        user_id INTEGER,
        item_id INTEGER,
        quantity INTEGER DEFAULT 1,
        FOREIGN KEY (user_id) REFERENCES users
                 (user_id),
        FOREIGN KEY
                 (item_id) REFERENCES items
                 (id),
        PRIMARY KEY (user_id, item_id)
        )""")

    conn.execute("""CREATE TABLE IF NOT EXISTS lotto_state
                    (
                        id INTEGER PRIMARY KEY,
                        winning_number INTEGER,
                        jackpot INTEGER,
                        last_bonus_date TEXT DEFAULT '',
                    )""")

    conn.commit()
    conn.close()
    print("Base de données initialisée avec succès.")


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
    """Récupère tout : infos du pari ET liste des mises."""
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


def get_lotto_state():
    """Récupère l'état du loto (numéro gagnant et jackpot)."""
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


def get_all_items_db():
    conn = get_connection()
    items = conn.execute("SELECT * FROM items").fetchall()
    conn.close()
    return items


def use_item_db(user_id, item_name):
    conn = get_connection()
    try:
        item = conn.execute("SELECT id FROM items WHERE name = ?", (item_name,)).fetchone()
        if not item:
            return False
        user_has_item = has_item(user_id, item_name)
        if user_has_item:
            conn.execute(
                "UPDATE inventory SET quantity = quantity - 1 WHERE user_id = ? AND item_id = ?",
                (user_id, item['id']))
        else:
            conn.execute("DELETE FROM inventory WHERE user_id = ? AND item_id = ?", (user_id, item['id']))
        conn.commit()
        return True
    finally:
        conn.close()


def add_item_to_inventory(user_id, item_name):
    conn = get_connection()
    try:
        item = conn.execute("SELECT id FROM items WHERE name = ?", (item_name,)).fetchone()
        if not item:
            return False
        inv_row = conn.execute("SELECT quantity FROM inventory WHERE user_id = ? AND item_id = ?",
                               (user_id, item['id'])).fetchone()
        if not inv_row:
            conn.execute("INSERT INTO inventory (user_id, item_id, quantity) VALUES (?, ?, 1)",
                         (user_id, item['id']))
        else:
            conn.execute("UPDATE inventory SET quantity = quantity + 1 WHERE user_id = ? AND item_id = ?",
                         (user_id, item['id']))
        conn.commit()
        return True

    except Exception as e:
        print(f"Erreur ajout inventaire : {e}")
        return False

    finally:
        conn.close()


def has_item(user_id, item_name, min_quantity=1):
    conn = get_connection()
    try:
        query = """
                SELECT inv.quantity
                FROM inventory inv
                         JOIN items it ON inv.item_id = it.id
                WHERE inv.user_id = ? \
                  AND it.name = ? \
                """
        row = conn.execute(query, (user_id, item_name)).fetchone()
        if not row:
            return False
        return row['quantity'] >= min_quantity
    except Exception as e:
        print(f"Erreur has_item : {e}")
        return False
    finally:
        conn.close()

def transfer_item_transaction(seller_id, buyer_id, item_name, price):
    conn = get_connection()
    try:
        item = conn.execute("SELECT id FROM items WHERE name = ?", (item_name,)).fetchone()
        if not item:
            return "NO_ITEM"
        item_id = item['id']
        seller_inv = conn.execute(
            "SELECT quantity FROM inventory WHERE user_id = ? AND item_id = ?",
            (seller_id, item_id)
        ).fetchone()
        if not seller_inv or seller_inv['quantity'] < 1:
            return "NO_ITEM"
        buyer_bal = conn.execute("SELECT balance FROM users WHERE user_id = ?", (buyer_id,)).fetchone()
        if not buyer_bal or buyer_bal['balance'] < price:
            return "NO_MONEY"
        use_item_db(seller_id, item_name)
        add_item_to_inventory(buyer_id, item_name)
        conn.commit()
        return "SUCCESS"
    except Exception as e:
        print(f"Erreur Trade: {e}")
        conn.rollback()
        return "ERROR"
    finally:
        conn.close()


def reset_user_limit(user_id, activity_name):
    conn = get_connection()
    conn.execute("UPDATE game_limits SET count = 0 WHERE user_id = ? AND game_name = ?",
                 (user_id, activity_name))
    conn.commit()
    conn.close()