import random

from src.database.balance import update_balance
from src.items.Item import Item, ItemRarity, ItemType


class RustyMagnet(Item):
    def __init__(self):
        super().__init__(
            name="Aimant Rouillé",
            price=30,
            description="🧲 Utilise-le pour trouver de la petite monnaie par terre.",
            rarity=ItemRarity.common,
            item_type=ItemType.consumable
        )

    async def use(self, ctx, **kwargs):
        found = random.randint(10, 80)
        update_balance(ctx.author.id, found)

        await ctx.send(f"🧲 Tu promènes l'aimant... *Cling !* Des pièces collent dessus ! Tu gagnes **${found}**.")
        return True


class Magnet(Item):
    def __init__(self):
        super().__init__(
            name="Aimant",
            price=50,
            description="🧲 Utilise-le pour trouver de la monnaie par terre.",
            rarity=ItemRarity.rare,
            item_type=ItemType.consumable
        )

    async def use(self, ctx, **kwargs):
        found = random.randint(20, 150)
        update_balance(ctx.author.id, found)

        await ctx.send(f"🧲 Tu promènes l'aimant... *Cling !* Des pièces collent dessus ! Tu gagnes **${found}**.")
        return True

class ElectricMagnet(Item):
    def __init__(self):
        super().__init__(
            name="Aimant électrique",
            price=500,
            description="🧲 Utilise-le pour trouver un max de monnaie par terre.",
            rarity=ItemRarity.epic,
            item_type=ItemType.consumable
        )

    async def use(self, ctx, **kwargs):
        found = random.randint(150, 1000)
        update_balance(ctx.author.id, found)

        await ctx.send(f"🧲 Tu promènes l'aimant... *Cling !* Des pièces collent dessus ! Tu gagnes **${found}**.")
        return True
