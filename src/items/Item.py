import discord
import random
from src.data_handling import (
    use_item_db, update_balance, reset_user_cooldown,
    get_connection
)

# --- LE REGISTRE GLOBAL ---
# C'est ici qu'on stockera le lien "Nom de l'item" -> "Classe Python"
ITEMS_REGISTRY = {}


class Item:
    def __init__(self, name, price, description, type="consumable"):
        self.name = name
        self.price = price
        self.description = description
        self.type = type

    def register(self):
        ITEMS_REGISTRY[self.name.lower()] = self
        conn = get_connection()
        try:
            conn.execute("""
                         INSERT OR IGNORE INTO items (name, price, description, effect_type)
                         VALUES (?, ?, ?, ?)
                         """, (self.name, self.price, self.description, self.type))
            conn.commit()
        except Exception as e:
            print(f"Erreur register item {self.name}: {e}")
        finally:
            conn.close()

    async def use(self, ctx, **kwargs):
        """
        Méthode à surcharger par les enfants.
        Retourne True si l'objet a été consommé avec succès.
        """
        await ctx.send(f"Cet objet ({self.name}) ne fait rien pour l'instant.")
        return False