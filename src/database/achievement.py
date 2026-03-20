from typing import Dict, Any
from src.database.db_handler import get_connection


def increment_stat(user_id: int, stat_name: str, amount: int = 1):
    conn = get_connection()
    try:
        query = f"""
            INSERT INTO user_stats (user_id, {stat_name}) 
            VALUES (?, ?)
            ON CONFLICT(user_id) 
            DO UPDATE SET {stat_name} = {stat_name} + ?;
        """
        conn.execute(query, (user_id, amount, amount))
        conn.commit()
    finally:
        conn.close()


def get_user_stats(user_id: int) -> dict:
    conn = get_connection()
    try:
        row = conn.execute("SELECT * FROM user_stats WHERE user_id = ?", (user_id,)).fetchone()
        stats: Dict[str, Any] = dict(row) if row else {}
        
        bal_row = conn.execute("SELECT balance FROM users WHERE user_id = ?", (user_id,)).fetchone()
        stats["balance"] = bal_row["balance"] if bal_row else 100
        
        pet_row = conn.execute("SELECT MAX(level) as max_lvl FROM user_pets WHERE user_id = ?", (user_id,)).fetchone()
        stats["max_pet_level"] = pet_row["max_lvl"] if pet_row and pet_row["max_lvl"] else 0
        
        pet_collection_rows = conn.execute("SELECT DISTINCT pet_type FROM user_pets WHERE user_id = ?", (user_id,)).fetchall()
        if pet_collection_rows:
            pet_collection = [row["pet_type"] for row in pet_collection_rows]
            from src.models.Pet import PETS_DB
            from src.items.Item import ItemRarity
            stats["collected_common_pets"] = sum(1 for p in pet_collection if PETS_DB.get(p, {}).get("rarity") == ItemRarity.common)
            stats["collected_rare_pets"] = sum(1 for p in pet_collection if PETS_DB.get(p, {}).get("rarity") == ItemRarity.rare)
            stats["collected_epic_pets"] = sum(1 for p in pet_collection if PETS_DB.get(p, {}).get("rarity") == ItemRarity.epic)
            stats["collected_legendary_pets"] = sum(1 for p in pet_collection if PETS_DB.get(p, {}).get("rarity") == ItemRarity.legendary)
            stats["collected_all_pets"] = len(pet_collection)
        else:
            stats["collected_common_pets"] = 0
            stats["collected_rare_pets"] = 0
            stats["collected_epic_pets"] = 0
            stats["collected_legendary_pets"] = 0
            stats["collected_all_pets"] = 0
            
        from src.database.pets import get_all_pet_ranks
        ranks = get_all_pet_ranks()
        stats["pet_ranks"] = [data["rank"] for data in ranks.values() if data["user_id"] == user_id]
        
        return stats
    finally:
        conn.close()


def check_and_unlock_achievements(user_id: int) -> list:
    stats = get_user_stats(user_id)
    if not stats:
        return []

    conn = get_connection()
    new_unlocks = []

    try:
        rows = conn.execute("SELECT achievement_id FROM user_achievements WHERE user_id = ?", (user_id,)).fetchall()
        unlocked_ids = [row["achievement_id"] for row in rows]

        from src.models.Achievement import Achievement

        for ach_id, ach_data in Achievement.registry.items():
            if ach_id not in unlocked_ids:
                if ach_data.check_func(stats):
                    conn.execute("INSERT INTO user_achievements (user_id, achievement_id) VALUES (?, ?)",
                                 (user_id, ach_id))
                    new_unlocks.append(ach_data)
        conn.commit()
        return new_unlocks
    finally:
        conn.close()


def format_achievements_unlocks(unlocks: list):
    import discord
    if not unlocks:
        return None
    
    desc = ""
    for ach in unlocks:
        desc += f"🎖️ **{ach.name}** {ach.emoji}\n*+{ach.glory} pts de Gloire*\n{ach.desc}\n\n"
        
    embed = discord.Embed(title="🎉 Nouveau Succès Débloqué !", description=desc, color=discord.Color.gold())
    return embed