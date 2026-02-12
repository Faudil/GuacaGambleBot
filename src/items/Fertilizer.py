from src.database.other import reset_user_limit
from src.items.Item import Item, ItemType, ItemRarity


class Fertilizer(Item):
    def __init__(self):
        super().__init__(
            name="Engrais",
            price=200,
            description="Accélère la pousés des récoltes ! Réinitialise le cooldown de !farm.",
            item_type=ItemType.consumable,
            rarity=ItemRarity.common
        )

    async def use(self, ctx, **kwargs):
        reset_user_limit(ctx.author.id, "farm")
        await ctx.send("☕ **Splash...** Les plantes reprennent en vigueur ! Ton `!farm` est de nouveau disponible.")
        return True
