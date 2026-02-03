import discord
import random
from src.data_handling import (
    get_connection
)
from src.globals import ITEMS_REGISTRY


class Item:
    def __init__(self, name, price, description, rarity="common", image_url=None):
        self.name = name.lower()
        self.price = price
        self.description = description
        self.rarity = rarity  # common, rare, epic, legendary, unique
        self.image_url = image_url
        self.type = "collectible"  # Par défau

    def get_discord_color(self):
        colors = {
            "common": discord.Color.light_grey(),
            "rare": discord.Color.blue(),
            "epic": discord.Color.purple(),
            "legendary": discord.Color.gold(),
            "unique": discord.Color.red()
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
                         """, (name, self.price, self.description, self.type))
            conn.commit()
        except Exception as e:
            print(f"Erreur register item {self.name}: {e}")
        finally:
            conn.close()

    async def use(self, ctx, **kwargs):
        await ctx.send(f"Cet objet ({self.name}) ne fait rien pour l'instant.")
        return False