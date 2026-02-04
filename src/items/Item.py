from enum import Enum

import discord

from src.database.db_handler import get_connection
from src.globals import ITEMS_REGISTRY

class ItemRarity(Enum):
    common = "common"
    rare = "rare"
    epic = "epic"
    legendary = "legendary"
    unique = "unique"

class ItemType(Enum):
    collectible = "collectible"
    consumable = "consumable"
    permanent = "permanent"
    resource = "resource"


class Item:
    def __init__(self, name, price, description, item_type: ItemType, rarity: ItemRarity=ItemRarity.common, image_url=None):
        self.name = name.lower()
        self.price = price
        self.description = description
        self.type = item_type
        self.rarity = rarity
        self.image_url = image_url

    def get_discord_color(self):
        colors = {
            ItemRarity.common: discord.Color.light_grey(),
            ItemRarity.rare: discord.Color.blue(),
            ItemRarity.epic: discord.Color.purple(),
            ItemRarity.legendary: discord.Color.gold(),
            ItemRarity.unique: discord.Color.red()
        }
        return colors.get(self.rarity, discord.Color.default())

    def register(self):
        name = self.name.lower()
        ITEMS_REGISTRY[name] = self
        conn = get_connection()
        try:
            conn.execute("""
                         INSERT OR IGNORE INTO items (name, price, description, effect_type)
                         VALUES (?, ?, ?, ?)
                         """, (name, self.price, self.description, self.type.value))
            conn.commit()
        except Exception as e:
            print(f"Erreur register item {self.name}: {e}")
        finally:
            conn.close()

    async def use(self, ctx, **kwargs):
        await ctx.send(f"Cet objet ({self.name}) ne fait rien pour l'instant.")
        return False