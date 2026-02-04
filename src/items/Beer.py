from src.database.other import reset_user_limit
from src.items.Item import Item, ItemType, ItemRarity


class Beer(Item):
    def __init__(self):
        super().__init__(
            name="Bière",
            price=50,
            description="La boisson du mineur ! Réinitialise le cooldown de !mine.",
            item_type=ItemType.consumable,
            rarity=ItemRarity.common
        )

    async def use(self, ctx, **kwargs):
        reset_user_limit(ctx.author.id, "mine")
        await ctx.send("☕ **Gloupgloup...** Ah elle rafraichit ! Ton `!mine` est de nouveau disponible.")
        return True