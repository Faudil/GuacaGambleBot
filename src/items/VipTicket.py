from src.database.other import reset_user_limit
from src.items.Item import Item, ItemRarity, ItemType


class VipTicket(Item):
    def __init__(self):
        super().__init__(
            name="ticket vip",
            price=100,
            description="Te donne une nouvelle chance, reset tes limites de casino et coinflip",
            rarity=ItemRarity.rare,
            item_type=ItemType.consumable
        )

    async def use(self, ctx, **kwargs):
        reset_user_limit(ctx.author.id, "coinflip")
        reset_user_limit(ctx.author.id, "slots")
        await ctx.send(" Tes `!coinflips` et `!casino` sont de nouveaux disponible ! Bonne chance !")
        return True