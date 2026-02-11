from src.database.other import reset_user_limit
from src.items.Item import Item, ItemRarity, ItemType


class CheatCoin(Item):
    def __init__(self):
        super().__init__(
            name="pièce truquée",
            price=200,
            description="Augmente ta chance. Passe ta probabilité de réussir ton pile ou face à 75%.",
            rarity=ItemRarity.legendary,
            item_type=ItemType.consumable
        )

    async def use(self, ctx, **kwargs):
        await ctx.send(f"🍀 **Zouuuuuu...** La chance est avec toi ! Passe ta probabilité de réussir ton pile ou face à 75%.")
        return False
