import random

from src.database.balance import update_balance
from src.items.Item import Item, ItemRarity, ItemType


class ScratchTicket(Item):
    def __init__(self):
        super().__init__(
            name="Ticket à Gratter",
            price=100,
            description="🎰 Grattez pour gagner jusqu'à 1000$ instantanément !",
            rarity=ItemRarity.common,
            item_type=ItemType.consumable
        )

    async def use(self, ctx, **kwargs):
        outcomes = [0, 25, 50, 100, 200, 1000]
        weights = [10, 10, 40, 30, 9, 1]
        win = random.choices(outcomes, weights=weights)[0]
        if win > 0:
            update_balance(ctx.author.id, win)
            await ctx.send(f"🎰 **Grattage...** Tu as gagné **${win}** !")
        else:
            await ctx.send(f"🎰 **Grattage...** Perdu ! Essayez encore.")
        return True
