import random
import discord
from src.items.Item import Item, ItemRarity, ItemType


class ThievesGlove(Item):
    def __init__(self):
        super().__init__(
            name="Thieve's glove",
            price=20,
            description="📜 .",
            rarity=ItemRarity.epic,
            item_type=ItemType.consumable
        )

    async def use(self, ctx, **kwargs):
        await ctx.send("❌ Fait `!rob @ta-victime` pour utiliser ces gants.")
        return False
