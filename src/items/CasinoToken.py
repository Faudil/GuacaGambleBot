from src.database.other import reset_user_limit
from src.items.Item import Item, ItemRarity, ItemType


class CasinoToken(Item):
    def __init__(self):
        super().__init__(
            name="jeton de casino",
            price=50,
            description="Te donne une nouvelle chance, reset ta limite de `!casino`",
            rarity=ItemRarity.rare,
            item_type=ItemType.consumable
        )

    async def use(self, ctx, **kwargs):
        reset_user_limit(ctx.author.id, "slots")
        await ctx.send(" Tes rolls de `!casino` sont de nouveaux disponible ! Bonne chance !")
        return True