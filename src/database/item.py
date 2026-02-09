from typing import Dict

from src.database.balance import update_balance
from src.database.db_handler import get_connection


def get_all_items_db():
    conn = get_connection()
    items = conn.execute("SELECT * FROM items").fetchall()
    conn.close()
    return items

def get_item_id_from_name(item_name) -> int:
    conn = get_connection()
    try:
        item = conn.execute("SELECT id FROM items WHERE name = ?", (item_name,)).fetchone()
        return item['id']
    finally:
        conn.close()


def remove_item_from_inventory(user_id, item_name, quantity: int = 1) -> bool:
    """
    :param quantity:
    :param user_id:
    :param item_name:
    :return: If the item has been removed
    """
    conn = get_connection()
    try:
        item = conn.execute("SELECT id FROM items WHERE name = ?", (item_name,)).fetchone()
        if not item:
            return False
        user_has_item = has_item(user_id, item_name, quantity)
        if user_has_item:
            conn.execute(
                "UPDATE inventory SET quantity = quantity - ? WHERE user_id = ? AND item_id = ?",
                (quantity, user_id, item['id']))
        else:
            conn.execute("DELETE FROM inventory WHERE user_id = ? AND item_id = ?", (user_id, item['id']))
        conn.commit()
        return user_has_item
    finally:
        conn.close()


def add_item_to_inventory(user_id, item_name, quantity: int = 1) -> bool:
    conn = get_connection()
    try:
        item = conn.execute("SELECT id FROM items WHERE name = ?", (item_name,)).fetchone()
        if not item:
            return False
        inv_row = conn.execute("SELECT quantity FROM inventory WHERE user_id = ? AND item_id = ?",
                               (user_id, item['id'])).fetchone()
        if not inv_row:
            conn.execute("INSERT INTO inventory (user_id, item_id, quantity) VALUES (?, ?, ?)",
                         (user_id, item['id'], quantity))
        else:
            conn.execute("UPDATE inventory SET quantity = quantity + ? WHERE user_id = ? AND item_id = ?",
                         (quantity, user_id, item['id']))
        conn.commit()
        return True

    except Exception as e:
        print(f"Erreur ajout inventaire : {e}")
        return False

    finally:
        conn.close()


def has_item(user_id, item_name: str, min_quantity=1):
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
        remove_item_from_inventory(seller_id, item_name)
        add_item_to_inventory(buyer_id, item_name)
        update_balance(seller_id, price)
        update_balance(buyer_id, -price)
        conn.commit()
        return "SUCCESS"
    except Exception as e:
        print(f"Erreur Trade: {e}")
        conn.rollback()
        return "ERROR"
    finally:
        conn.close()


def get_item_name_by_id(item_id: int) -> str | None:
    conn = get_connection()
    try:
        row = conn.execute("SELECT name FROM items WHERE id = ?", (item_id,)).fetchone()
        if row:
            return row['name']
        return None
    finally:
        conn.close()


def get_all_user_inventory(user_id) -> list[Dict]:
    conn = get_connection()
    result = []
    try:
        rows = conn.execute("""
                            SELECT i.id, i.name, inv.quantity
                            FROM inventory inv
                                     JOIN items i ON inv.item_id = i.id
                            WHERE inv.user_id = ?
                            ORDER BY i.price DESC
                            """, (user_id,)).fetchall()
        for row in rows:
            result.append({
                "id": row['id'],
                "name": row['name'],
                "quantity": row['quantity']
            })
        return result
    finally:
        conn.close()
        return result