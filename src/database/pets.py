from src.database.db_handler import get_connection
from src.models.Pet import Pet


def insert_new_pet(pet: Pet):
    conn = get_connection()
    try:
        cursor = conn.execute("""
                              INSERT INTO user_pets
                              (user_id, pet_type, nickname, level, xp, max_hp, hp, atk, defense, speed, dge, acc,
                               crit_c, crit_d, elo, bonus, is_active)
                              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                              """, (
                                  pet.user_id, pet.pet_type, pet.nickname, pet.level, pet.xp, pet.max_hp, pet.hp,
                                  pet.atk, pet.defense, pet.speed, pet.dge, pet.acc, pet.crit_c, pet.crit_d,
                                  pet.elo, pet.bonus_type, pet.is_active
                              ))
        conn.commit()
        pet.id = cursor.lastrowid
        return pet
    finally:
        conn.close()


def update_pet(pet: Pet):
    conn = get_connection()
    try:
        conn.execute("""
                     UPDATE user_pets
                     SET level=?,
                         nickname=?,
                         xp=?,
                         max_hp=?,
                         hp=?,
                         atk=?,
                         defense=?,
                         speed=?,
                         dge=?,
                         acc=?,
                         crit_c=?,
                         crit_d=?,
                         spc_c=?,
                         trs_lvl=?,
                         elo=?,
                         bonus=?,
                         food_eaten=?
                     WHERE id = ?
                     """, (
                         pet.level, pet.nickname, pet.xp, pet.max_hp, pet.hp, pet.atk, pet.defense, pet.speed,
                         pet.dge, pet.acc, pet.crit_c, pet.crit_d, pet.spc_c, pet.trs_lvl, pet.elo, pet.bonus_type, pet.food_eaten,
                         pet.id
                     ))
        conn.commit()
    finally:
        conn.close()


def get_active_pet(user_id) -> Pet:
    conn = get_connection()
    try:
        row = conn.execute("SELECT * FROM user_pets WHERE user_id = ? AND is_active = 1", (user_id,)).fetchone()
        if row:
            return Pet.from_db(dict(row))
        return None
    finally:
        conn.close()


def get_all_pets(user_id) -> list[Pet]:
    conn = get_connection()
    try:
        rows = conn.execute("SELECT * FROM user_pets WHERE user_id = ?", (user_id,)).fetchall()
        return [Pet.from_db(dict(row)) for row in rows]
    finally:
        conn.close()


def set_active_pet(user_id, pet_id):
    conn = get_connection()
    try:
        conn.execute("UPDATE user_pets SET is_active = 0 WHERE user_id = ?", (user_id,))
        cursor = conn.execute("UPDATE user_pets SET is_active = 1 WHERE id = ? AND user_id = ?", (pet_id, user_id))
        conn.commit()
        return cursor.rowcount > 0
    finally:
        conn.close()


def get_pet_by_id(pet_id: int) -> Pet:
    conn = get_connection()
    try:
        row = conn.execute("SELECT * FROM user_pets WHERE id = ?", (pet_id,)).fetchone()
        if row:
            return Pet.from_db(dict(row))
        return None
    finally:
        conn.close()


def transfer_pet(pet_id: int, new_owner_id: int):
    conn = get_connection()
    try:
        conn.execute("UPDATE user_pets SET user_id = ?, is_active = 0 WHERE id = ?", (new_owner_id, pet_id))
        conn.commit()
    finally:
        conn.close()
