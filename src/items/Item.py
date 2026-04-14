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




from src.utils.i18n import t, get_item_name, get_item_description

class Item:
    def __init__(self, name, price, description, item_type: ItemType, rarity: ItemRarity=ItemRarity.common, image_url=None, pet_effet=None):
        self.name = name.lower()
        self.price = price
        self.description = description
        self.id = -1
        self.type = item_type
        self.rarity = rarity
        self.image_url = image_url
        self.pet_effect = pet_effet

    def display_name(self, lang: str = 'fr'):
        return get_item_name(self.name, lang)

    def display_description(self, lang: str = 'fr'):
        return get_item_description(self.name, lang)

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
            item = conn.execute("SELECT id FROM items WHERE name = ?", (self.name,)).fetchone()
            self.id = item["id"]
        except Exception as e:
            print(f"Error register item {self.name}: {e}")
        finally:
            conn.close()

    async def use(self, ctx, **kwargs):
        from src.database.settings import get_language
        lang = get_language(ctx.guild.id if ctx.guild else None)
        await ctx.send(t("items.base_use_msg", lang, item_name=self.display_name(lang)))
        return False