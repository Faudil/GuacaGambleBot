from src.database.db_handler import get_connection
from datetime import datetime, timedelta
import json

def get_user_housing(user_id):
    """Récupère les informations de logement d'un utilisateur."""
    conn = get_connection()
    try:
        row = conn.execute("SELECT * FROM user_housing WHERE user_id = ?", (user_id,)).fetchone()
        if row:
            return dict(row)
        return None
    finally:
        conn.close()

def buy_house(user_id, house_type):
    """Achète ou remplace la maison d'un utilisateur."""
    conn = get_connection()
    try:
        conn.execute("""
            INSERT INTO user_housing (user_id, house_type, level, last_collected, stored_items)
            VALUES (?, ?, 1, ?, '{}')
            ON CONFLICT(user_id) DO UPDATE SET 
                house_type = EXCLUDED.house_type, 
                level = 1, 
                last_collected = EXCLUDED.last_collected,
                stored_items = '{}',
                under_construction = NULL,
                finish_time = NULL
        """, (user_id, house_type, datetime.now().isoformat()))
        conn.commit()
    finally:
        conn.close()

def upgrade_house(user_id):
    """Augmente le niveau de la maison actuelle."""
    conn = get_connection()
    try:
        conn.execute("UPDATE user_housing SET level = level + 1 WHERE user_id = ?", (user_id,))
        conn.commit()
    finally:
        conn.close()

def add_housing_upgrade(user_id, upgrade_id):
    """Ajoute une amélioration spécifique (meuble, extension)."""
    conn = get_connection()
    try:
        conn.execute("INSERT OR IGNORE INTO user_housing_upgrades (user_id, upgrade_id) VALUES (?, ?)", (user_id, upgrade_id))
        conn.commit()
    finally:
        conn.close()

def get_housing_upgrades(user_id):
    """Récupère toutes les améliorations achetées par un utilisateur."""
    conn = get_connection()
    try:
        rows = conn.execute("SELECT upgrade_id FROM user_housing_upgrades WHERE user_id = ?", (user_id,)).fetchall()
        return [row['upgrade_id'] for row in rows]
    finally:
        conn.close()

def update_last_collected(user_id):
    """Met à jour la date de dernière récolte de revenus passifs."""
    conn = get_connection()
    try:
        conn.execute("UPDATE user_housing SET last_collected = ? WHERE user_id = ?", (datetime.now().isoformat(), user_id))
        conn.commit()
    finally:
        conn.close()

def rename_house(user_id, name):
    """Change le nom personnalisé de la maison."""
    conn = get_connection()
    try:
        conn.execute("UPDATE user_housing SET custom_name = ? WHERE user_id = ?", (name, user_id))
        conn.commit()
    finally:
        conn.close()

def set_house_color(user_id, color):
    """Change la couleur personnalisée de l'embed de la maison."""
    conn = get_connection()
    try:
        conn.execute("UPDATE user_housing SET custom_color = ? WHERE user_id = ?", (color, user_id))
        conn.commit()
    finally:
        conn.close()

def start_construction(user_id, upgrade_id, hours):
    """Démarre un projet de construction."""
    conn = get_connection()
    try:
        finish_time = (datetime.now() + timedelta(hours=hours)).isoformat()
        conn.execute("""
            UPDATE user_housing 
            SET under_construction = ?, finish_time = ? 
            WHERE user_id = ?
        """, (upgrade_id, finish_time, user_id))
        conn.commit()
    finally:
        conn.close()

def complete_construction(user_id):
    """Finalise une construction terminée."""
    conn = get_connection()
    try:
        row = conn.execute("SELECT under_construction FROM user_housing WHERE user_id = ?", (user_id,)).fetchone()
        if row and row['under_construction']:
            upgrade_id = row['under_construction']
            conn.execute("INSERT OR IGNORE INTO user_housing_upgrades (user_id, upgrade_id) VALUES (?, ?)", (user_id, upgrade_id))
            conn.execute("UPDATE user_housing SET under_construction = NULL, finish_time = NULL WHERE user_id = ?", (user_id,))
            conn.commit()
            return True
        return False
    finally:
        conn.close()

def get_crowns(user_id):
    """Récupère le solde de Crowns d'un utilisateur."""
    conn = get_connection()
    try:
        row = conn.execute("SELECT crowns FROM users WHERE user_id = ?", (user_id,)).fetchone()
        return row['crowns'] if row else 0
    finally:
        conn.close()

def update_crowns(user_id, amount):
    """Modifie le solde de Crowns d'un utilisateur."""
    conn = get_connection()
    try:
        conn.execute("UPDATE users SET crowns = crowns + ? WHERE user_id = ?", (amount, user_id))
        conn.commit()
    finally:
        conn.close()

def get_stored_items(user_id):
    """Récupère les objets produits stockés dans la maison."""
    conn = get_connection()
    try:
        row = conn.execute("SELECT stored_items FROM user_housing WHERE user_id = ?", (user_id,)).fetchone()
        if row and row['stored_items']:
            return json.loads(row['stored_items'])
        return {}
    finally:
        conn.close()

def update_stored_items(user_id, items_dict):
    """Met à jour le stock d'objets produits de la maison."""
    conn = get_connection()
    try:
        conn.execute("UPDATE user_housing SET stored_items = ? WHERE user_id = ?", (json.dumps(items_dict), user_id))
        conn.commit()
    finally:
        conn.close()

from src.housing_data import HOUSES

def get_user_capacities(user_id):
    """Calcule les capacités maximales d'inventaire et de pets d'un utilisateur."""
    conn = get_connection()
    try:
        user = conn.execute("SELECT extra_inv_slots, extra_pet_slots FROM users WHERE user_id = ?", (user_id,)).fetchone()
        housing = conn.execute("SELECT house_type FROM user_housing WHERE user_id = ?", (user_id,)).fetchone()
        
        extra_inv = user['extra_inv_slots'] if user else 0
        extra_pets = user['extra_pet_slots'] if user else 0
        
        inv_bonus = 0
        pet_bonus = 0
        
        if housing and housing['house_type'] in HOUSES:
            house = HOUSES[housing['house_type']]
            inv_bonus = house.get('inventory_bonus', 0)
            pet_bonus = house.get('pet_slots_bonus', 0)
            
        return {
            "max_inv": 50 + inv_bonus + extra_inv,
            "max_pets": 3 + pet_bonus + extra_pets
        }
    finally:
        conn.close()

def is_inventory_full(user_id):
    capacities = get_user_capacities(user_id)
    
    conn = get_connection()
    try:
        # Calculer la quantité totale d'objets
        total_items = conn.execute("SELECT SUM(quantity) as total FROM inventory WHERE user_id = ?", (user_id,)).fetchone()
        current_count = total_items['total'] if total_items['total'] else 0
        
        return current_count >= capacities['max_inv'], current_count, capacities['max_inv']
    finally:
        conn.close()

def can_add_pet(user_id):
    capacities = get_user_capacities(user_id)
    
    conn = get_connection()
    try:
        pet_count = conn.execute("SELECT COUNT(*) as count FROM user_pets WHERE user_id = ?", (user_id,)).fetchone()
        current_count = pet_count['count'] if pet_count else 0
        
        return current_count < capacities['max_pets'], current_count, capacities['max_pets']
    finally:
        conn.close()

def add_extra_slots(user_id, inv_slots=0, pet_slots=0):
    conn = get_connection()
    try:
        conn.execute("""
            UPDATE users 
            SET extra_inv_slots = extra_inv_slots + ?, 
                extra_pet_slots = extra_pet_slots + ? 
            WHERE user_id = ?
        """, (inv_slots, pet_slots, user_id))
        conn.commit()
    finally:
        conn.close()
